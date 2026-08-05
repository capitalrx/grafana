import { type DataFrame, DataFrameView, type TraceSpanRow } from '@grafana/data';

import transformTraceData from '../components/model/transform-trace-data';
import { type Trace, type TraceProcess, type TraceResponse } from '../components/types/trace';

export function transformDataFrames(frames?: DataFrame | DataFrame[]): Trace | null {
  const frameList = toFrameArray(frames);
  if (frameList.length === 0) {
    return null;
  }
  let data: TraceResponse | null =
    frameList.length === 1 && frameList[0].fields.length === 1
      ? // For backward compatibility when we sent whole json response in a single field/value
        frameList[0].fields[0].values[0]
      : transformTraceDataFrames(frameList);

  if (!data) {
    return null;
  }
  return transformTraceData(data);
}

function toFrameArray(frames?: DataFrame | DataFrame[]): DataFrame[] {
  if (!frames) {
    return [];
  }
  return Array.isArray(frames) ? frames : [frames];
}

export function transformTraceDataFrame(frame: DataFrame): TraceResponse | null {
  return transformTraceDataFrames([frame]);
}

export function transformTraceDataFrames(frames: DataFrame[]): TraceResponse | null {
  const processes: Record<string, TraceProcess> = {};
  const spans: TraceResponse['spans'] = [];
  let traceID: string | undefined;

  for (let frameIndex = 0; frameIndex < frames.length; frameIndex++) {
    const view = new DataFrameView<TraceSpanRow>(frames[frameIndex]);

    for (let i = 0; i < view.length; i++) {
      const span = view.get(i);
      if (!span.spanID) {
        return null;
      }
      if (!processes[span.spanID]) {
        processes[span.spanID] = {
          serviceName: span.serviceName,
          serviceNamespace: span.serviceNamespace,
          tags: Array.isArray(span.serviceTags) ? span.serviceTags : [],
        };
      }
      if (traceID === undefined) {
        traceID = span.traceID;
      }
    }

    view.toArray().forEach((s, index) => {
      const references = [];
      if (s.parentSpanID) {
        references.push({ refType: 'CHILD_OF' as const, spanID: s.parentSpanID, traceID: s.traceID });
      }
      if (s.references) {
        references.push(...s.references.map((reference) => ({ refType: 'FOLLOWS_FROM' as const, ...reference })));
      }
      spans.push({
        ...s,
        duration: s.duration * 1000,
        startTime: s.startTime * 1000,
        processID: s.spanID,
        flags: 0,
        references,
        logs: s.logs?.map((l) => ({ ...l, timestamp: l.timestamp * 1000 })) || [],
        dataFrameRowIndex: index,
        dataFrameIndex: frames.length > 1 ? frameIndex : undefined,
      });
    });
  }

  if (traceID === undefined) {
    return null;
  }

  return {
    traceID,
    processes,
    spans,
  };
}
