/* tslint:disable */
import errorHandler from "errorhandler";

import app from "./app";
import { logger } from "./utils/logger";

/**
 * error handler. provides full stack - remove for production
 */
if (process.env.ENVIRONMENT != "production") {
  app.use(errorHandler());
}

/**
 * start express server.
 */
const server = app.listen(app.get("port"), () => {
  logger.log(
    "info",
    `app is running at http://localhost:${app.get("port")} in ${app.get(
      "env",
    )} mode`,
    { foo: "bar" },
  );
  logger.log("info", "Press CTRL-C to stop\n", { foo: "baz" });
});

export default server;
