import { View } from 'sourcegraph'

import { createDriverForTest, Driver } from '@sourcegraph/shared/src/testing/driver'
import { afterEachSaveScreenshotIfFailed } from '@sourcegraph/shared/src/testing/screenshotReporter'

import { createWebIntegrationTestContext, WebIntegrationTestContext } from '../context'
import { percySnapshotWithVariants } from '../utils'

import {
    BACKEND_INSIGHTS,
    CODE_STATS_INSIGHT_LANG_USAGE,
    INSIGHT_VIEW_TEAM_SIZE,
    INSIGHT_VIEW_TYPES_MIGRATION,
} from './utils/insight-mock-data'
import { overrideGraphQLExtensions } from './utils/override-graphql-with-extensions'

describe('[VISUAL] Code insights page', () => {
    let driver: Driver
    let testContext: WebIntegrationTestContext

    for (const browser of ['chrome', 'firefox'] as const) {

        before(async () => {
            driver = await createDriverForTest({ browser, sourcegraphBaseUrl: 'https://sourcegraph.test:3443'})
        })

        after(() => driver?.close())

        beforeEach(async function () {
            testContext = await createWebIntegrationTestContext({
                driver,
                currentTest: this.currentTest!,
                directory: __dirname,
            })
        })

        afterEachSaveScreenshotIfFailed(() => driver.page)
        afterEach(() => testContext?.dispose())

        it('is styled correctly with back-end insights', async () => {
            overrideGraphQLExtensions({
                testContext,
                overrides: {
                    /**
                     * Mock back-end insights with standard gql API handler.
                     * */
                    Insights: () => ({ insights: { nodes: BACKEND_INSIGHTS } }),
                },
            })

            await driver.page.goto(driver.sourcegraphBaseUrl + '/insights')
            await driver.page.waitForSelector('[data-testid="line-chart__content"] svg circle')

            await percySnapshotWithVariants(driver.page, 'Code insights page with back-end insights only')
        })

        it('is styled correctly with search-based insights ', async () => {
            overrideGraphQLExtensions({
                testContext,

                /**
                 * Since search insight and code stats insight are working via user/org
                 * settings. We have to mock them by mocking user settings and provide
                 * mock data - mocking extension work.
                 * */
                userSettings: {
                    'searchInsights.insight.graphQLTypesMigration': {},
                    'searchInsights.insight.teamSize': {},
                },
                insightExtensionsMocks: {
                    'searchInsights.insight.teamSize': INSIGHT_VIEW_TEAM_SIZE,
                    'searchInsights.insight.graphQLTypesMigration': INSIGHT_VIEW_TYPES_MIGRATION,
                },
                overrides: {
                    /**
                     * Mock back-end insights with standard gql API handler.
                     * */
                    Insights: () => ({ insights: { nodes: [] } }),
                },
            })

            await driver.page.goto(driver.sourcegraphBaseUrl + '/insights')
            await driver.page.waitForSelector('[data-testid="line-chart__content"] svg circle')

            await percySnapshotWithVariants(driver.page, 'Code insights page with search-based insights only')
        })

        it('is styled correctly with errored insight', async () => {
            overrideGraphQLExtensions({
                testContext,

                /**
                 * Since search insight and code stats insight are working via user/org
                 * settings. We have to mock them by mocking user settings and provide
                 * mock data - mocking extension work.
                 * */
                userSettings: {
                    'searchInsights.insight.graphQLTypesMigration': {},
                    'searchInsights.insight.teamSize': {},
                },
                insightExtensionsMocks: {
                    'searchInsights.insight.teamSize': ({ message: 'Error message', name: 'hello' } as unknown) as View,
                    'searchInsights.insight.graphQLTypesMigration': INSIGHT_VIEW_TYPES_MIGRATION,
                },
                overrides: {
                    /**
                     * Mock back-end insights with standard gql API handler.
                     * */
                    Insights: () => ({ insights: { nodes: [] } }),
                },
            })

            await driver.page.goto(driver.sourcegraphBaseUrl + '/insights')
            await driver.page.waitForSelector('[data-testid="searchInsights.insight.teamSize.insightsPage insight error"]')

            await percySnapshotWithVariants(driver.page, 'Code insights page with search-based errored insight')
        })

        it('is styled correctly with all types of insight', async () => {
            overrideGraphQLExtensions({
                testContext,

                /**
                 * Since search insight and code stats insight are working via user/org
                 * settings. We have to mock them by mocking user settings and provide
                 * mock data - mocking extension work.
                 * */
                userSettings: {
                    'searchInsights.insight.graphQLTypesMigration': {},
                    'searchInsights.insight.teamSize': {},
                    'codeStatsInsights.insight.langUsage': {},
                },
                insightExtensionsMocks: {
                    'codeStatsInsights.insight.langUsage': CODE_STATS_INSIGHT_LANG_USAGE,
                    'searchInsights.insight.teamSize': INSIGHT_VIEW_TEAM_SIZE,
                    'searchInsights.insight.graphQLTypesMigration': INSIGHT_VIEW_TYPES_MIGRATION,
                },
                overrides: {
                    /**
                     * Mock back-end insights with standard gql API handler.
                     * */
                    Insights: () => ({ insights: { nodes: BACKEND_INSIGHTS } }),
                },
            })

            await driver.page.goto(driver.sourcegraphBaseUrl + '/insights')
            await driver.page.waitForSelector('[data-testid="line-chart__content"] svg circle')

            await percySnapshotWithVariants(driver.page, 'Code insights page with all types of insight')
        })
    }
})
