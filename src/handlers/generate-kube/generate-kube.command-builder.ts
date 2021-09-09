import yargs from 'yargs';

export type generateKubeArgs = {typeOfConfig: string} &
{clusterName: string} &
{namespace: string} &
{labels: string[]} &
{customPort: number} &
{outputFile: string} &
{environmentId: string}

export function generateKubeCmdBuilder(yargs: yargs.Argv<{}>) : yargs.Argv<generateKubeArgs> {
    return yargs
        .positional('typeOfConfig', {
            type: 'string',
            choices: ['kubeConfig', 'kubeYaml']

        }).positional('clusterName', {
            type: 'string',
            default: null
        }).option('namespace', {
            type: 'string',
            default: '',
            demandOption: false
        }).option('labels', {
            type: 'array',
            default: [],
            demandOption: false
        }).option('customPort', {
            type: 'number',
            default: -1,
            demandOption: false
        }).option('outputFile', {
            type: 'string',
            demandOption: false,
            alias: 'o',
            default: null
        })
        .option('environmentId', {
            type: 'string',
            default: null
        })
        .example('$0 generate kubeYaml testcluster', '')
        .example('$0 generate kubeConfig', '')
        .example('$0 generate kubeYaml --labels testkey:testvalue', '');
}