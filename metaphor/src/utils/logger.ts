import winston from "winston";

/**
 * create logger supported by dd apm auto injection
 */
export const logger = winston.createLogger({
  transports: [new winston.transports.Console()],
});
