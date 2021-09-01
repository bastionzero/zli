import yargs from 'yargs';

export function connectCmdBuilder (
    yargs : yargs.Argv<{}>,
    targetTypeChoices : string[]) :
    yargs.Argv<{targetString: string;} & {targetType: string;}>
{
    return yargs
        .positional('targetString', {
            type: 'string',
        })
        .option(
            'targetType',
            {
                type: 'string',
                choices: targetTypeChoices,
                demandOption: false,
                alias: 't'
            },
        )
        .example('connect ssm-user@neat-target', 'SSM connect example, uniquely named ssm target')
        .example('connect --targetType dynamic ssm-user@my-dat-config', 'DAT connect example with a DAT configuration whose name is my-dat-config');
}