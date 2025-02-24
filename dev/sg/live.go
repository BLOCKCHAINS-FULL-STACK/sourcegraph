package main

import (
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/dev/sg/root"
	"github.com/sourcegraph/sourcegraph/lib/output"
)

type environment struct {
	Name string
	URL  string
}

var environments = []environment{
	{Name: "dot-com", URL: "https://sourcegraph.com"},
	{Name: "k8s", URL: "https://k8s.sgdev.org"},
}

func environmentNames() []string {
	var names []string
	for _, e := range environments {
		names = append(names, e.Name)
	}
	return names
}

func getEnvironment(name string) (result environment, found bool) {
	for _, e := range environments {
		if e.Name == name {
			result = e
			found = true
		}
	}

	return result, found
}

func printDeployedVersion(e environment) error {
	pending := out.Pending(output.Linef("", output.StylePending, "Fetching newest version on %q...", e.Name))

	resp, err := http.Get(e.URL + "/__version")
	if err != nil {
		pending.Complete(output.Linef(output.EmojiFailure, output.StyleWarning, "Failed: %s", err))
		return err
	}
	defer resp.Body.Close()

	pending.Complete(output.Linef(output.EmojiSuccess, output.StyleSuccess, "Done"))

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	elems := strings.Split(string(body), "_")
	if len(elems) != 3 {
		return fmt.Errorf("unknown format of /__version response: %q", body)
	}

	buildDate := elems[1]
	buildSha := elems[2]

	pending = out.Pending(output.Line("", output.StylePending, "Running 'git fetch' to update list of commits..."))
	_, err = runGitCmd("fetch", "-q")
	if err != nil {
		pending.Complete(output.Linef(output.EmojiFailure, output.StyleWarning, "Failed: %s", err))
		return err
	}
	pending.Complete(output.Linef(output.EmojiSuccess, output.StyleSuccess, "Done updating list of commits"))

	log, err := runGitCmd("log", "--oneline", "-n", "20", `--pretty=format:%h|%ar|%an|%s`, "origin/main")
	if err != nil {
		pending.Complete(output.Linef(output.EmojiFailure, output.StyleWarning, "Failed: %s", err))
		return err
	}

	out.Write("")
	line := output.Linef(
		output.EmojiLightbulb, output.StyleLogo,
		"Live on %q: %s%s%s %s(built on %s)",
		e.Name, output.StyleBold, buildSha, output.StyleReset, output.StyleLogo, buildDate,
	)
	out.WriteLine(line)

	out.Write("")

	var shaFound bool
	for _, logLine := range strings.Split(log, "\n") {
		elems := strings.SplitN(logLine, "|", 4)
		sha := elems[0]
		timestamp := elems[1]
		author := elems[2]
		message := elems[3]

		var emoji string = "  "
		var style output.Style = output.StylePending
		if sha[0:len(buildSha)] == buildSha {
			emoji = "🚀"
			style = output.StyleLogo
			shaFound = true
		}

		line := output.Linef(emoji, style, "%s (%s, %s): %s", sha, timestamp, author, message)
		out.WriteLine(line)
	}

	if !shaFound {
		line := output.Linef(output.EmojiWarning, output.StyleWarning, "Deployed SHA %s not found in last 20 commits on origin/main :(", buildSha)
		out.WriteLine(line)
	}

	return nil
}

func runGitCmd(args ...string) (string, error) {
	repoRoot, err := root.RepositoryRoot()
	if err != nil {
		return "", err
	}

	cmd := exec.Command("git", args...)
	cmd.Env = []string{
		// Don't use the system wide git config.
		"GIT_CONFIG_NOSYSTEM=1",
		// And also not any other, because they can mess up output, change defaults, .. which can do unexpected things.
		"GIT_CONFIG=/dev/null",
	}
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.Wrapf(err, "'git %s' failed: %s", strings.Join(args, " "), out)
	}

	return string(out), nil
}
