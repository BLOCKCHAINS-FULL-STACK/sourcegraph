# Updating a single-image Sourcegraph instance (`sourcegraph/server`)

This document describes the exact changes needed to update a single-node Sourcegraph instance.

A new version of Sourcegraph is released every month (with patch releases in between, released as needed). Check the [Sourcegraph blog](https://about.sourcegraph.com/blog) or the site admin updates page to learn about updates. We actively maintain the two most recent monthly releases of Sourcegraph.

> :warning: **Regardless of your deployment type:** upgrade one version at a time, e.g. v3.26 --> v3.27 --> v3.28.
> <br>(Note that patches, e.g. vX.X.4 vs. vX.X.5 do not have to be adopted when moving between vX.X versions)

**Always refer to this page before upgrading Sourcegraph,** as it comprehensively describes the steps needed to upgrade, and any manual migration steps you must perform.

## 3.26 -> 3.27

> Warning: ⚠️ Sourcegraph 3.27 now requires **Postgres 12+**.

If you are using an external database, [upgrade your database](https://docs.sourcegraph.com/admin/postgres#upgrading-external-postgresql-instances) to Postgres 12 or above prior to upgrading Sourcegraph. If you are using the embedded database, [prepare your data for migration](https://docs.sourcegraph.com/admin/postgres#upgrading-single-node-docker-deployments) prior to upgrading Sourcegraph.

## 3.25 -> 3.26

## Standard upgrade procedure

To update, just use the newer `sourcegraph/server:N.N.N` Docker image (where `N.N.N` is the version number) in place of the older one, using the same Docker volumes. Your server's data will be migrated automatically if needed.

You can always find the version number of the latest release at [docs.sourcegraph.com](https://docs.sourcegraph.com) in the `docker run` command's image tag.

- As a precaution, before updating, we recommend backing up the contents of the Docker volumes used by Sourcegraph.
- If you need a HA deployment, use the [Kubernetes cluster deployment option](https://github.com/sourcegraph/deploy-sourcegraph).
- There is currently no automated way to downgrade to an older version after you have updated. [Contact support](https://about.sourcegraph.com/contact) for help.


> NOTE: ⚠️ From **3.27** onwards we will only support PostgreSQL versions **starting from 12**.

## 3.24 -> 3.25

- Go `1.15` introduced changes to SSL/TLS connection validation which requires certificates to include a `SAN`. This field was not included in older certificates and clients relied on the `CN` field. You might see an error like `x509: certificate relies on legacy Common Name field`. We recommend that customers using Sourcegraph with an external database and and connecting to it using SSL/TLS check whether the certificate is up to date.
  - AWS RDS customers please reference [AWS' documentation on updating the SSL/TLS certificate](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/UsingWithRDS.SSL-certificate-rotation.html) for steps to rotate your certificate.

## 3.20 -> 3.21

This release introduces a second database instance, `codeintel-db`. If you have configured Sourcegraph with an external database, then update the `CODEINTEL_PG*` environment variables to point to a new external database as described in the [external database documentation](../external_services/postgres.md). Again, these must not point to the same database or the Sourcegraph instance will refuse to start.

### If you wish to keep existing LSIF data

> WARNING: **Do not upgrade out of the 3.21.x release branch** until you have seen the log message indicating the completion of the LSIF data migration, or verified that the `/lsif-storage/dbs` directory on the precise-code-intel-bundle-manager volume is empty. Otherwise, you risk data loss for precise code intelligence.

If you had LSIF data uploaded prior to upgrading to 3.21.0, there is a background migration that moves all existing LSIF data into the `codeintel-db` upon upgrade. Once this process completes, the `/lsif-storage/dbs` directory on the precise-code-intel-bundle-manager volume should be empty, and the bundle manager should print the following log message:

> Migration to Postgres has completed. All existing LSIF bundles have moved to the path /lsif-storage/db-backups and can be removed from the filesystem to reclaim space.

**Wait for the above message to be printed in `docker logs precise-code-intel-bundle-manager` before upgrading to the next Sourcegraph version**.

### Turn off database secrets encryption

> WARNING: Please delete the secret key file `/var/lib/sourcegraph/token` inside the container before attempting to upgrade to 3.21.x.

In Sourcegraph version 3.20, we would automatically generate a secret key file (`/var/lib/sourcegraph/token`) inside the container for encrypting secrets stored in the database. However, it is not yet ready for general use and format of the secret key file might change. Therefore, it is best to delete the secret key file (`/var/lib/sourcegraph/token`) and turn off the database secrets encryption.

## 3.16 -> 3.17

- There was [a bug](https://github.com/sourcegraph/sourcegraph/issues/11618) in the `sourcegraph/server:3.28.0` release that caused the version displayed on the `site-admin/update` page to be `0.0.0+dev` instead of `3.17.0`. This issue [was fixed](https://github.com/sourcegraph/sourcegraph/pull/11633) in the `3.17.2` release. We recommend that you avoid this issue by upgrading past `3.17.0` to `3.17.2` using the [Standard upgrade procedure](#Standard-upgrade-procedure) listed below.

## Standard upgrade procedure

To update, just use the newer `sourcegraph/server:N.N.N` Docker image (where `N.N.N` is the version number) in place of the older one, using the same Docker volumes. Your server's data will be migrated automatically if needed.

You can always find the version number of the latest release at [docs.sourcegraph.com](https://docs.sourcegraph.com) in the `docker run` command's image tag.

- As a precaution, before updating, we recommend backing up the contents of the Docker volumes used by Sourcegraph.
- If you need a HA deployment, use the [Kubernetes cluster deployment option](https://github.com/sourcegraph/deploy-sourcegraph).
- There is currently no automated way to downgrade to an older version after you have updated. [Contact support](https://about.sourcegraph.com/contact) for help.
