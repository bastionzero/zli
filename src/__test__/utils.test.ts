import { parseTargetString, parseTargetType } from '../utils';

test('valid targetType strings', () => {
    const validSSMTargetTypeStrings = [
        'ssh',
        'ssm',
        'SSH', // caps don't matter
        'SSM',
        'SsH',
        'sSM'
    ];
    validSSMTargetTypeStrings.forEach(t => expect(parseTargetType(t)).toBeDefined());
});

test('invalid targetType strings', () => {
    const validSSMTargetTypeStrings = [
        '123123',
        'ssmA', // too long
        'sssm', // too long
        'SSHssm',
        'SuSHiMi', // SSH and SSM embedded
        'mss'
    ];
    validSSMTargetTypeStrings.forEach(t => expect(parseTargetType(t)).toBeUndefined());
});

test('valid targetStrings', () => {
    const validSSMTargetStrings = [
        'ssm-user@neat-test',
        '_ssm-user@coolBeans',
        'ssm-user$@97d4d916-33f8-478e-9e6c-1091662ccaf0', // valid $ in unixname
        'ssm-user@neat-test:/hello', // valid path
        'ssm-user@coolBeans:::', // everything after first colon ignored
        'ssm-user@97d4d916-33f8-478e-9e6c-1091662ccaf0:asdfjl; asdfla;sd',
        '97d4d916-33f8-478e-9e6c-1091662ccaf0:asdfjl; asdfla;sd'
    ];
    validSSMTargetStrings.forEach(t => expect(parseTargetString(t)).toBeDefined());
});

test('invalid targetStrings', () => {
    const validSSMTargetStrings = [
        'ssm$-user@neat-test',  // invalid unix username, $ wrong place
        'ss..er@neat-test:/hello', // invalid characters in unix username
        'ssm-user@:97d4d916-33f8-478e-9e6c-1091662ccaf0', // colon wrong place
        'ss!!!r@whatsUp!Word:/cool' // invalid character in target name
    ];
    validSSMTargetStrings.forEach(t => expect(parseTargetString(t)).toBeUndefined());
});