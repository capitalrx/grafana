import { createDataFrame } from '@grafana/data';

import { transformDataFrames, transformTraceDataFrame } from './transform';

describe('transformTraceDataFrame()', () => {
  const fields = [
    { name: 'traceID', values: ['trace1'] },
    { name: 'operationName', values: ['operation1'] },
    { name: 'kind', values: ['server'] },
    { name: 'tags', values: [[{ key: 'key1', value: 'value1' }]] },
  ];

  it('should return transformed data', () => {
    const dummyDataFrame = createDataFrame({
      fields: fields.concat([...fields, { name: 'spanID', values: ['span1'] }]),
    });
    expect(transformTraceDataFrame(dummyDataFrame)).toEqual({
      processes: { span1: { serviceName: undefined, serviceNamespace: undefined, tags: [] } },
      spans: [
        {
          dataFrameRowIndex: 0,
          duration: NaN,
          flags: 0,
          kind: 'server',
          logs: [],
          operationName: 'operation1',
          processID: 'span1',
          references: [],
          spanID: 'span1',
          startTime: NaN,
          tags: [{ key: 'key1', value: 'value1' }],
          traceID: 'trace1',
        },
      ],
      traceID: 'trace1',
    });
  });

  it('should return null for any span without a spanID', () => {
    const dummyDataFrame = createDataFrame({
      fields: fields,
    });
    expect(transformTraceDataFrame(dummyDataFrame)).toEqual(null);
  });

  it('should map serviceNamespace from DataFrame into process when present', () => {
    const frameWithNamespace = createDataFrame({
      fields: [
        { name: 'traceID', values: ['trace1'] },
        { name: 'spanID', values: ['span1'] },
        { name: 'operationName', values: ['GET /api'] },
        { name: 'serviceName', values: ['cart-service'] },
        { name: 'serviceNamespace', values: ['production'] },
        { name: 'kind', values: ['server'] },
        { name: 'tags', values: [[]] },
      ],
    });
    const result = transformTraceDataFrame(frameWithNamespace);
    expect(result).not.toBeNull();
    expect(result!.processes['span1']).toEqual({
      serviceName: 'cart-service',
      serviceNamespace: 'production',
      tags: [],
    });
  });
});

describe('transformDataFrames() with multiple frames', () => {
  it('combines spans from multiple frames into a single trace and tags each span with its frame index', () => {
    const frameA = createDataFrame({
      refId: 'A',
      fields: [
        { name: 'traceID', values: ['trace1'] },
        { name: 'spanID', values: ['span1'] },
        { name: 'operationName', values: ['GET /api'] },
        { name: 'startTime', values: [1000] },
        { name: 'duration', values: [10] },
      ],
    });
    const frameB = createDataFrame({
      refId: 'B',
      fields: [
        { name: 'traceID', values: ['trace1'] },
        { name: 'spanID', values: ['span2'] },
        { name: 'operationName', values: ['GET /downstream'] },
        { name: 'startTime', values: [1005] },
        { name: 'duration', values: [3] },
      ],
    });

    const trace = transformDataFrames([frameA, frameB]);
    expect(trace).not.toBeNull();
    expect(trace!.spans.map((s) => ({ spanID: s.spanID, dataFrameIndex: s.dataFrameIndex }))).toEqual([
      { spanID: 'span1', dataFrameIndex: 0 },
      { spanID: 'span2', dataFrameIndex: 1 },
    ]);
  });

  it('does not set dataFrameIndex when only a single frame is provided', () => {
    const frame = createDataFrame({
      refId: 'A',
      fields: [
        { name: 'traceID', values: ['trace1'] },
        { name: 'spanID', values: ['span1'] },
        { name: 'operationName', values: ['GET /api'] },
        { name: 'startTime', values: [1000] },
        { name: 'duration', values: [10] },
      ],
    });

    const trace = transformDataFrames([frame]);
    expect(trace).not.toBeNull();
    expect(trace!.spans[0].dataFrameIndex).toBeUndefined();
  });
});
