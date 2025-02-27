import React, { useCallback, useState, useEffect } from 'react'

import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import { Link } from '@sourcegraph/shared/src/components/Link'
import { asError, ErrorLike, isErrorLike } from '@sourcegraph/shared/src/util/errors'
import { isDefined, keyExistsIn } from '@sourcegraph/shared/src/util/types'

import { ErrorAlert } from '../../../components/alerts'
import { queryExternalServices } from '../../../components/externalServices/backend'
import { AddExternalServiceOptions } from '../../../components/externalServices/externalServices'
import { PageTitle } from '../../../components/PageTitle'
import { Scalars, ExternalServiceKind, ListExternalServiceFields } from '../../../graphql-operations'
import { SourcegraphContext } from '../../../jscontext'
import { eventLogger } from '../../../tracking/eventLogger'
import { UserExternalServicesOrRepositoriesUpdateProps } from '../../../util'
import { githubRepoScopeRequired } from '../cloud-ga'

import { CodeHostItem } from './CodeHostItem'

type AuthProvider = SourcegraphContext['authProviders'][0]
type AuthProvidersByKind = Partial<Record<ExternalServiceKind, AuthProvider>>

export interface UserAddCodeHostsPageProps extends UserExternalServicesOrRepositoriesUpdateProps {
    user: { id: Scalars['ID']; tags: string[] }
    codeHostExternalServices: Record<string, AddExternalServiceOptions>
    routingPrefix: string
    context: Pick<SourcegraphContext, 'authProviders'>
}

type ServicesByKind = Partial<Record<ExternalServiceKind, ListExternalServiceFields>>
type Status = undefined | 'loading' | ServicesByKind | ErrorLike

const isServicesByKind = (status: Status): status is ServicesByKind =>
    typeof status === 'object' && Object.keys(status).every(key => keyExistsIn(key, ExternalServiceKind))

export const ifNotNavigated = (callback: () => void, waitMS: number = 2000): void => {
    let timeoutID = 0
    let willNavigate = false

    const unloadListener = (): void => {
        willNavigate = true
    }

    window.addEventListener('unload', unloadListener)

    timeoutID = window.setTimeout(() => {
        // if we waited waitMS and the navigation didn't happen - run the callback
        if (!willNavigate) {
            // cleanup
            window.removeEventListener('unload', unloadListener)
            window.clearTimeout(timeoutID)

            return callback()
        }
    }, waitMS)
}

export const UserAddCodeHostsPage: React.FunctionComponent<UserAddCodeHostsPageProps> = ({
    user,
    codeHostExternalServices,
    routingPrefix,
    context,
    onUserExternalServicesOrRepositoriesUpdate,
}) => {
    const [statusOrError, setStatusOrError] = useState<Status>()

    const fetchExternalServices = useCallback(async () => {
        setStatusOrError('loading')

        const { nodes: fetchedServices } = await queryExternalServices({
            namespace: user.id,
            first: null,
            after: null,
        }).toPromise()

        const services: ServicesByKind = fetchedServices.reduce<ServicesByKind>((accumulator, service) => {
            // backend constraint - non-admin users have only one external service per ExternalServiceKind
            accumulator[service.kind] = service
            return accumulator
        }, {})

        setStatusOrError(services)

        const repoCount = fetchedServices.reduce((sum, codeHost) => sum + codeHost.repoCount, 0)
        onUserExternalServicesOrRepositoriesUpdate(fetchedServices.length, repoCount)
    }, [user.id, onUserExternalServicesOrRepositoriesUpdate])

    useEffect(() => {
        eventLogger.logViewEvent('UserSettingsCodeHostConnections')
    }, [])

    useEffect(() => {
        fetchExternalServices().catch(error => {
            setStatusOrError(asError(error))
        })
    }, [fetchExternalServices])

    const getGitHubUpdateAuthBanner = (needsUpdate: boolean): JSX.Element | null =>
        needsUpdate ? (
            <div className="alert alert-info mb-4" role="alert" key="add-repos">
                Update your GitHub code host connection to search private code with Sourcegraph.
            </div>
        ) : null

    const getAddReposBanner = (services: string[]): JSX.Element | null =>
        services.length > 0 ? (
            <div className="alert alert-success mb-4" role="alert" key="add-repos">
                Connected with {services.join(', ')}. Next,{' '}
                <Link className="alert-link" to={`${routingPrefix}/repositories/manage`}>
                    add your repositories →
                </Link>
            </div>
        ) : null

    const getErrorAndSuccessBanners = (status: Status): (JSX.Element | null)[] => {
        const servicesWithProblems = []
        const notYetSyncedServiceNames = []
        let updateAuthRequired = false

        // check if services are fetched
        if (isServicesByKind(status)) {
            const services = Object.values(status).filter(isDefined)

            for (const service of services) {
                // if service has warnings or errors
                if (service.warning || service.lastSyncError) {
                    servicesWithProblems.push(service)
                    continue
                }

                // if service is not synced yet or has a "sync now" timestamp
                // "sync now" timestamp is always less then the epoch time

                // don't display user name in service name
                const serviceName = service.displayName.split(' ')[0]

                if (!service?.lastSyncAt) {
                    notYetSyncedServiceNames.push(serviceName)
                } else {
                    const lastSyncTime = new Date(service.lastSyncAt)
                    const epochTime = new Date(0)

                    if (lastSyncTime < epochTime) {
                        notYetSyncedServiceNames.push(serviceName)
                    }
                }

                // If we have a GitHub service, check whether we need to prompt the user to
                // update their scope
                if (service.kind === 'GITHUB') {
                    updateAuthRequired = githubRepoScopeRequired(user.tags, service.grantedScopes)
                }
            }
        }

        return [
            ...servicesWithProblems.map(getServiceWarningFragment),
            getAddReposBanner(notYetSyncedServiceNames),
            getGitHubUpdateAuthBanner(updateAuthRequired),
        ]
    }

    const addNewService = useCallback(
        (service: ListExternalServiceFields): void => {
            if (isServicesByKind(statusOrError)) {
                setStatusOrError({ ...statusOrError, [service.kind]: service })
            }
        },
        [statusOrError]
    )

    const handleError = useCallback((error: ErrorLike): void => setStatusOrError(error), [])

    const getServiceWarningFragment = ({ id, displayName }: ListExternalServiceFields): JSX.Element => (
        <div className="alert alert-danger my-4" key={id}>
            <strong className="align-middle">Could not connect to {displayName}.</strong>
            <span className="align-middle">
                {' '}
                Please remove {displayName} code host connection and try again to restore the connection.
            </span>
        </div>
    )

    // auth providers by service type
    const authProvidersByKind = context.authProviders.reduce((accumulator: AuthProvidersByKind, provider) => {
        if (provider.authenticationURL) {
            accumulator[provider.serviceType.toLocaleUpperCase() as ExternalServiceKind] = provider
        }
        return accumulator
    }, {})

    const navigateToAuthProvider = useCallback(
        (kind: ExternalServiceKind): void => {
            const authProvider = authProvidersByKind[kind]

            if (authProvider) {
                eventLogger.log('UserAttemptConnectCodeHost', { kind })
                window.location.assign(
                    `${authProvider.authenticationURL as string}&redirect=${
                        window.location.href
                    }&op=createCodeHostConnection`
                )
            }
        },
        [authProvidersByKind]
    )

    return (
        <div className="user-code-hosts-page">
            <PageTitle title="Code host connections" />
            <div className="mb-4">
                <div className="d-flex justify-content-between align-items-center mb-3">
                    <h2 className="mb-0">Code host connections</h2>
                </div>
                <p className="text-muted">
                    Connect with your code hosts. Then,{' '}
                    <Link className="text-primary" to={`${routingPrefix}/repositories/manage`}>
                        add repositories
                    </Link>{' '}
                    to search with Sourcegraph.
                </p>
            </div>

            {/* display external service errors and success banners */}
            {getErrorAndSuccessBanners(statusOrError)}

            {/* display other errors, e.g. network errors */}
            {isErrorLike(statusOrError) && (
                <ErrorAlert error={statusOrError} prefix="Code host action error" icon={false} />
            )}

            {codeHostExternalServices && isServicesByKind(statusOrError) ? (
                <>
                    <ul className="list-group">
                        {Object.entries(codeHostExternalServices).map(([id, { kind, defaultDisplayName, icon }]) =>
                            authProvidersByKind[kind] ? (
                                <li key={id} className="list-group-item">
                                    <CodeHostItem
                                        service={isServicesByKind(statusOrError) ? statusOrError[kind] : undefined}
                                        user={user}
                                        kind={kind}
                                        name={defaultDisplayName}
                                        navigateToAuthProvider={navigateToAuthProvider}
                                        icon={icon}
                                        onDidAdd={addNewService}
                                        onDidRemove={fetchExternalServices}
                                        onDidError={handleError}
                                    />
                                </li>
                            ) : null
                        )}
                    </ul>
                </>
            ) : (
                <div className="d-flex justify-content-center">
                    <LoadingSpinner className="icon-inline" />
                </div>
            )}
        </div>
    )
}
