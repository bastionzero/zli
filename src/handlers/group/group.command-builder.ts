import yargs from 'yargs';

export type groupArgs = {add: boolean} &
{delete: boolean} &
{groupName: string} &
{policyName: string} &
{json: boolean}

export function groupCmdBuilder(yargs: yargs.Argv<{}>) : yargs.Argv<groupArgs> {
    return yargs
        .option(
            'add',
            {
                type: 'boolean',
                demandOption: false,
                alias: 'a',
                implies: ['groupName', 'policyName']
            }
        )
        .option(
            'delete',
            {
                type: 'boolean',
                demandOption: false,
                alias: 'd',
                implies: ['groupName', 'policyName']
            }
        )
        .conflicts('add', 'delete')
        .positional('groupName',
            {
                type: 'string',
                default: null,
                demandOption: false,
            }
        )
        .positional('policyName',
            {
                type: 'string',
                default: null,
                demandOption: false,
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
        .example('$0 group --json', 'List all groups, output as json, pipeable')
        .example('$0 group --add cool-policy engineering-group', 'Adds the engineering-group IDP group to cool-policy policy')
        .example('$0 group -d cool-policy engineering-group', 'Deletes the engineering-group IDP group from the cool-policy policy');
}