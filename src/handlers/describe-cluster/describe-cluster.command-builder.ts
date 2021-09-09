import yargs from 'yargs';

type describeClusterArgs = {clusterName : string};

export function describeClusterCmdBuilder(yargs: yargs.Argv<{}>) : yargs.Argv<describeClusterArgs> {
    return yargs
        .positional('clusterName', {
            type: 'string',
        })
        .example('$0 describe-cluster test-cluster', '');
}