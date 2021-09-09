import yargs from 'yargs';

type connectArgs = {targetString: string} & {targetType: string}

export function connectCmdBuilder (yargs : yargs.Argv<{}>,targetTypeChoices : string[]) : yargs.Argv<connectArgs>
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
        .example('$0 connect ssm-user@neat-target', 'SSM connect example, uniquely named ssm target')
        .example('$0 connect --targetType dynamic ssm-user@my-dat-config', 'DAT connect example with a DAT configuration whose name is my-dat-config');
}