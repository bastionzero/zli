import yargs from 'yargs';

export function disconnectCmdBuilder(yargs : yargs.Argv<{}>) : yargs.Argv<{}> {
    return yargs
        .example('$0 disconnect', 'Disconnect a local Zli Daemon');
}