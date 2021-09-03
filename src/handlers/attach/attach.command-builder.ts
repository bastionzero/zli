import yargs from "yargs";

export function attachCmdBuilder(yargs : yargs.Argv<{}>) :
yargs.Argv<{connectionId : string}> {
    return yargs
        .positional('connectionId', {
            type: 'string',
        })
        .example('attach d5b264c7-534c-4184-a4e4-3703489cb917', 'attach example, unique connection id');
}