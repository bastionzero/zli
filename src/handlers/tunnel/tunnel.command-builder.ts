import yargs from "yargs";

export function tunnelCmdBuilder (yargs : yargs.Argv<{}>) :
yargs.Argv<{tunnelString: string;} & {customPort: number;}> {
    return yargs
        .positional('tunnelString', {
            type: 'string',
            default: null,
            demandOption: false,
        }).option('customPort', {
            type: 'number',
            default: -1,
            demandOption: false
        })
        .example('tunnel admin@neat-cluster', 'Connect to neat-cluster as the admin Kube RBAC role');
}