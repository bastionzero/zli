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

/**
 * QuickstartSSMTarget represents an SSH host that has successfully been added
 * to BastionZero as an SSM target
 */
export interface QuickstartSSMTarget {
    ssmTarget: SsmTargetSummary;
    sshHost: ValidSSHHost;
}

/**
 * ValidSSHHost encapsulates the information needed to start an SSH connection
 */
export interface ValidSSHHost {
    name: string;
    hostIp: string;
    port: number;
    username: string;
    identityFile: string;
}

/**
 * ValidSSHHostAndConfig associates the SSH configuration used by the
 * ssh2-promise library to start an SSH connection to some valid SSH host
 */
export interface ValidSSHHostAndConfig {
    config: SSHConfig;
    sshHost: ValidSSHHost;
}

/**
 * RegistrableSSHHost associates an environment ID to use during the
 * registration process to some valid SSH host
 */
export interface RegistrableSSHHost {
    host: ValidSSHHostAndConfig;
    envId: string;
}

/**
 * MissingHostNameParseError represents a failure to include the HostName
 * parameter in the parsed SSH config file
 */
export type MissingHostNameParseError = {
    error: 'missing_host_name'
};

/**
 * MissingPortParseError represents a failure to include the Port parameter in
 * the parsed SSH config file
 */
export type MissingPortParseError = {
    error: 'missing_port'
}

/**
 * MissingUserParseError represents a failure to include the User parameter in
 * the parsed SSH config file
 */
export type MissingUserParseError = {
    error: 'missing_user'
}

/**
 * MissingIdentityFileParseError represents a failure to include the
 * IdentityFile (path to SSH key) parameter in the parsed SSH config file
 */
export type MissingIdentityFileParseError = {
    error: 'missing_identity_file'
}

/**
 * SSHConfigParseError is a sum type of all possible parse errors when reading
 * an SSH config file
 */
export type SSHConfigParseError =
    | MissingHostNameParseError
    | MissingPortParseError
    | MissingUserParseError
    | MissingIdentityFileParseError

/**
 * InvalidSSHHost represents an SSH host which does not have all the parameters
 * that are necessary for an SSH connection to be made to it.
 */
export interface InvalidSSHHost {
    /**
     * Partially valid SSH host (all parameters are possibly undefined)
     */
    incompleteValidSSHHost: ValidSSHHost;
    /**
     * A list of all parse errors that must be resolved for
     * incompleteValidSSHHost to be considered valid
     */
    parseErrors: SSHConfigParseError[]
}