/* tslint:disable */
const { initTracer: initJaegerTracer } = require("jaeger-client");

// this is the url to try next, DNS debugging kubernetes
// collectorEndpoint: 'http://jaeger-operator-jaeger-collector.default.svc.cluster.local:14268/api/traces',

module.exports.initTracer = (serviceName: string) => {
  const config = {
    serviceName,
    sampler: {
      type: "const",
      param: 1,
    },
    reporter: {
      collectorEndpoint:
        "http://jaeger-operator-jaeger-collector:14268/api/traces",
      logSpans: true,
    },
  };
  const options = {
    logger: {
      info(msg: string) {
        console.log("INFO ", msg);
      },
      error(msg: string) {
        console.log("ERROR", msg);
      },
    },
  };

  return initJaegerTracer(config, options);
};
