/* tslint:disable */
import bodyParser from "body-parser";
import compression from "compression"; // compresses requests
import express from "express";
import fs from "fs";
import path from "path";

import { logger } from "./utils/logger";
import * as homeView from "./views/home";

// Controllers (route handlers)

// Create Express server
const app = express();

function delay(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

app.set("views", path.join(__dirname, "views")); // this is the folder where we keep our pug files
app.set("view engine", "pug"); // we use the engine pug, mustache or EJS work great too

// serves up static files from the public folder. Anything in public/ will just be served up as the file it is
app.use(express.static(path.join(__dirname, "public")));

// Express configuration
app.set("port", process.env.PORT || 3000);
app.use(compression());
app.use(bodyParser.json());
app.use(bodyParser.urlencoded({ extended: true }));

/**
 * Primary app routes.
 */
app.get("/", function(req, res) {
  res.render("index", {
    appName: "metaphor",
    companyName: "kubefirst",
    chartVersion: process.env.CHART_VERSION,
    dockerTag: process.env.DOCKER_TAG,
    secretOne: process.env.SECRET_ONE,
    secretTwo: process.env.SECRET_TWO,
    configOne: process.env.CONFIG_ONE,
    configTwo: process.env.CONFIG_TWO,
  });
});

app.get("/performance", async (req, res) => {
  const sleepTime = Math.floor(Math.random() * 2 * 1000);

  fs.readFile("test.txt", "utf8", function(err, data) {
    if (err) {
      logger.info("error", err);
    }
    logger.info("info", data);
  });
  await delay(sleepTime);

  res.send({ hello: "world", sleepTime });
});

app.get("/healthz", homeView.status);
app.get("/kill", homeView.kill);

export default app;
