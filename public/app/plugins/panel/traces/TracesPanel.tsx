import { css } from '@emotion/css';
import { useMemo, useRef } from 'react';
import { useAsync } from 'react-use';

import { type TraceSearchProps, type Field, type LinkModel, type PanelProps } from '@grafana/data';
import { Trans } from '@grafana/i18n';
import { getDataSourceSrv } from '@grafana/runtime';
import { type DataSourceRef } from '@grafana/schema';
import { TraceView } from 'app/features/explore/TraceView/TraceView';
import { type SpanLinkFunc } from 'app/features/explore/TraceView/components/types/links';
import { transformDataFrames } from 'app/features/explore/TraceView/utils/transform';

import { replaceSearchVariables } from '../../../features/explore/TraceView/useSearch';

const styles = {
  wrapper: css({
    height: '100%',
    overflow: 'scroll',
  }),
};

export interface TracesPanelOptions {
  /** Fallback datasource identity for when data is provided via SceneDataNode and PanelData.request is unavailable. Required for extension points like grafana/traceview/header/actions. */
  datasource?: DataSourceRef;
  createSpanLink?: SpanLinkFunc;
  focusedSpanId?: string;
  createFocusSpanLink?: (traceId: string, spanId: string) => LinkModel<Field>;
  spanFilters?: TraceSearchProps;
  hideHeaderDetails?: boolean;
}

export const TracesPanel = ({ data, options, replaceVariables }: PanelProps<TracesPanelOptions>) => {
  const topOfViewRef = useRef<HTMLDivElement>(null);
  const traceProp = useMemo(() => transformDataFrames(data.series), [data.series]);
  const dataSource = useAsync(async () => {
    const uid = data.request?.targets?.[0]?.datasource?.uid ?? options?.datasource?.uid;

    if (!uid) {
      return undefined;
    }

    return await getDataSourceSrv().get(uid);
  });
  const dataFrameDataSourceUids = useMemo(
    () =>
      data.series.map((frame) => {
        const target = data.request?.targets?.find((t) => t.refId === frame.refId);
        return target?.datasource?.uid ?? data.request?.targets?.[0]?.datasource?.uid ?? options?.datasource?.uid;
      }),
    [data.series, data.request, options?.datasource]
  );

  if (!data || !data.series.length || !traceProp) {
    return (
      <div className="panel-empty">
        <p>
          <Trans i18nKey="traces.traces-panel.no-data-found-in-response">No data found in response</Trans>
        </p>
      </div>
    );
  }

  return (
    <div className={styles.wrapper}>
      <div ref={topOfViewRef}></div>
      <TraceView
        dataFrames={data.series}
        dataFrameDataSourceUids={dataFrameDataSourceUids}
        scrollElementClass={styles.wrapper}
        traceProp={traceProp}
        datasource={dataSource.value}
        topOfViewRef={topOfViewRef}
        createSpanLink={options.createSpanLink}
        focusedSpanId={options.focusedSpanId}
        createFocusSpanLink={options.createFocusSpanLink}
        spanFilters={replaceSearchVariables(replaceVariables, options.spanFilters)}
        timeRange={data.timeRange}
        hideHeaderDetails={options.hideHeaderDetails}
      />
    </div>
  );
};
