import yargs from "yargs";

export type describeClusterArgs = {clusterName : string};

export function describeClusterCmdBuilder(yargs: yargs.Argv<{}>) : yargs.Argv<describeClusterArgs> {
    return yargs
        .positional('clusterName', {
            type: 'string',
        })
        .example('status test-cluster', '');
}