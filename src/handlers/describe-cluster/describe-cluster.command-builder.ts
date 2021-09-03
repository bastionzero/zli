import yargs from "yargs";

export function describeClusterCmdBuilder(yargs: yargs.Argv<{}>) :
yargs.Argv<{clusterName : string}> {
    return yargs
        .positional('clusterName', {
            type: 'string',
        })
        .example('status test-cluster', '');
}