import SSHConfig from 'ssh2-promise/lib/sshConfig';
import { SsmTargetSummary } from '../ssm-target/ssm-target.types';

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

// QuickstartSSMTarget represents an SSH host that has successfully been added
// to BastionZero as an SSM target
export interface QuickstartSSMTarget {
    ssmTarget: SsmTargetSummary;
    sshHost: ValidSSHHost;
}

// ValidSSHHost encapsulates the information needed to start an SSH connection
export interface ValidSSHHost {
    name: string;
    hostIp: string;
    port: number;
    username: string;
    identityFile: string;
}

// ValidSSHHostAndConfig encapsulates the SSH configuration used by the SSH2-promise library to start an SSH connection to a valid SSH host
export interface ValidSSHHostAndConfig {
    config: SSHConfig;
    sshHost: ValidSSHHost;
}

// RegistrableSSHHost represents a valid SSH host that can be registered as it
// contains the environment ID to use during registration process
export interface RegistrableSSHHost {
    host: ValidSSHHostAndConfig;
    envId: string;
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