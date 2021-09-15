/* tslint:disable */
import { Request, Response } from "express";

// const { initTracer } = require("../utils/tracer");
// const tracer = initTracer("property-service");

/**
 * GET /healthz
 * health check route for kubernetes
 */
export const status = (req: Request, res: Response) => {
  res.send({
    status:
      "ok",
  });
};

/**
 * GET /kill
 * a route to kill the node app
 */
export const kill = (req: Request, res: Response) => {
  setTimeout(() => {
    process.exit(1);
  },         3000);
};
