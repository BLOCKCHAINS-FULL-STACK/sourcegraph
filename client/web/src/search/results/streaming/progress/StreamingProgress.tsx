import * as React from 'react'

import { Progress, StreamingResultsState } from '../../../stream'

import { StreamingProgressCount } from './StreamingProgressCount'
import { StreamingProgressSkippedButton } from './StreamingProgressSkippedButton'

export interface StreamingProgressProps {
    state: StreamingResultsState
    progress: Progress
    showTrace?: boolean
    onSearchAgain: (additionalFilters: string[]) => void
}

export const StreamingProgress: React.FunctionComponent<StreamingProgressProps> = props => (
    <>
        <StreamingProgressCount {...props} />
        <StreamingProgressSkippedButton {...props} />
    </>
)
