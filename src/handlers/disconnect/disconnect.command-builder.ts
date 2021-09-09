import yargs from 'yargs';

export function disconnectCmdBuilder(yargs : yargs.Argv<{}>) : yargs.Argv<{}> {
    return yargs
        .example('disconnect', 'Disconnect a local Zli Daemon');
}