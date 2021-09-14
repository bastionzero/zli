import SSHConfig from 'ssh2-promise/lib/sshConfig';

// Interface types for SSHConfig parsing package
export interface SSHHostConfig {
    param: string;
    value: string;
}
export interface SSHConfigHostBlock {
    param: string;
    value: string;
    config: SSHHostConfig[]
}

// SSHHost encapsulates the information needed to start an SSH connection
export interface ValidSSHHost {
    name: string;
    hostIp: string;
    port: number;
    username: string;
    identityFile: string;
}

export interface ValidSSHConfig {
    config: SSHConfig;
    sshHostName: string;
}

export type MissingHostNameParseError = {
    error: 'missing_host_name'
};

export type MissingPortParseError = {
    error: 'missing_port'
}

export type MissingUserParseError = {
    error: 'missing_user'
}

export type MissingIdentityFileParseError = {
    error: 'missing_identity_file'
}

export type SSHConfigParseError =
    | MissingHostNameParseError
    | MissingPortParseError
    | MissingUserParseError
    | MissingIdentityFileParseError

export interface InvalidSSHHost {
    incompleteValidSSHHost: ValidSSHHost;
    parseErrors: SSHConfigParseError[]
}