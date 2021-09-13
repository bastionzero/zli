import yargs from 'yargs';
import { generateBashCmdBuilder } from './generate-bash.command-builder';

// Tests that code in .check() in generateBashCmdBuilder() does not mess up
// mutual exclusion check on --targetName and --targetNameScheme flags
test.each([
    [['--targetName', 'foo', '--targetNameScheme', 'time']],
    [['--targetName', 'foo', '       --targetNameScheme', 'time']],
    [['--targetName', 'foo', '--targetNameScheme=time']],
    [['--targetName', 'foo', '       --targetNameScheme=time']],
])('check mutually exclusive error is thrown with processArgs: %s', (processArgs) => {
    // Simulate passing in arguments for yargs and store validation error
    let validationErr: Error;
    yargs.command('generate-bash', '', (yargs) => generateBashCmdBuilder(processArgs, yargs), () => { })
        .parse(['generate-bash', ...processArgs], {}, (err) => {
            validationErr = err;
        });

    // Assert that mutual exclusion error is thrown
    expect(validationErr).not.toBeNull();
    expect(validationErr.message).toEqual('Arguments targetNameScheme and targetName are mutually exclusive');
});

// Tests that code in .check() in generateBashCmdBuilder() correctly permits the
// --targetName flag to be passed by itself without any validation errors (e.g.
// mutual exclusion error)
test.each([
    [['--targetName', 'foo']],
    [['-n', 'foo']],
])('check no validation error is thrown with processArgs: %s', (processArgs) => {
    // Simulate passing in arguments for yargs and store validation error
    let validationErr: Error;
    yargs.command('generate-bash', '', (yargs) => generateBashCmdBuilder(processArgs, yargs), () => { })
        .parse(['generate-bash', ...processArgs], {}, (err) => {
            validationErr = err;
        });

    // Assert that no validation error is thrown
    expect(validationErr).toBeNull();
});