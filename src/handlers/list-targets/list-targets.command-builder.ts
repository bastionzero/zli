import yargs from 'yargs';

export type listTargetsArgs = {targetType: string} &
{env: string} &
{name: string} &
{status: string[]} &
{detail: boolean} &
{showId: boolean} &
{json: boolean}

export function listTargetsCmdBuilder(yargs: yargs.Argv<{}>, targetTypeChoices: string[], targetStatusChoices: string[]) : yargs.Argv<listTargetsArgs> {
    return yargs
        .option(
            'targetType',
            {
                type: 'string',
                choices: targetTypeChoices,
                demandOption: false,
                alias: 't'
            }
        )
        .option(
            'env',
            {
                type: 'string',
                demandOption: false,
                alias: 'e'
            }
        )
        .option(
            'name',
            {
                type: 'string',
                demandOption: false,
                alias: 'n'
            }
        )
        .option(
            'status',
            {
                type: 'string',
                array: true,
                choices: targetStatusChoices,
                alias: 'u'
            }
        )
        .option(
            'detail',
            {
                type: 'boolean',
                default: false,
                demandOption: false,
                alias: 'd'
            }
        )
        .option(
            'showId',
            {
                type: 'boolean',
                default: false,
                demandOption: false,
                alias: 'i'
            }
        )
        .option(
            'json',
            {
                type: 'boolean',
                default: false,
                demandOption: false,
                alias: 'j',
            }
        )
        .example('$0 lt -t ssm', 'List all SSM targets only')
        .example('$0 lt -i', 'List all targets and show unique ids')
        .example('$0 lt -e prod --json --silent', 'List all targets targets in prod, output as json, pipeable');
}