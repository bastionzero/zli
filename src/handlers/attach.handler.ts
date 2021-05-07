import { ConnectionService } from "../../src/http.service/http.service";
import { ConfigService } from "../../src/config.service/config.service";
import { Logger } from "../../src/logger.service/logger";
import { createShellHandler } from "./create-shell.handler";
import { ConnectionState } from "../../src/http.service/http.service.types";
import { cleanExit } from "./clean-exit.handler";

export async function attachHandler(
    configService: ConfigService,
    logger: Logger,
    connectionId: string
){
    const connectionService = new ConnectionService(configService, logger);
    const connectionSummary = await connectionService.GetConnection(connectionId);
    if (connectionSummary.state !== ConnectionState.Open){
        logger.error(`Connection ${connectionId} is not open`);
        await cleanExit(1, logger);
    }
    await createShellHandler(configService, logger, connectionSummary.serverType, connectionSummary.serverId, connectionId);
}