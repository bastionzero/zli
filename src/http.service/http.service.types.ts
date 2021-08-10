import { ReadStream } from 'fs';
import { SessionState, SsmTargetStatus, KubeClusterStatus, TargetType } from '../types';

export interface CreateSessionRequest {
    displayName?: string;
    connectionsToOpen: ConnectionsToOpen[];
}

export interface CreateSessionResponse {
    sessionId: string;
}

export interface CloseSessionRequest {
    sessionId: string;
}

export interface CloseSessionResponse {
}

export interface ListSessionsRequest {
}

export interface ListSessionsResponse {
    sessions: SessionDetails[];
}

export interface SessionDetails {
    id: string;
    displayName: string;
    timeCreated: number;
    state: SessionState,
    connections: ConnectionSummary[]
}

export interface ConnectionsToOpen {
      serverId: string;
      connectionType: TargetType,
      count: number
}

export enum ConnectionState {
    Open = 'Open',
    Closed = 'Closed',
    Error = 'Error'
}

export interface CreateConnectionRequest {
    sessionId: string;
    serverId: string;
    serverType: TargetType;
    username: string;
}

export interface CreateConnectionResponse {
    connectionId: string;
}

export interface CloseConnectionRequest {
    connectionId: string;
}

export interface CloseConnectionResponse {
}

export interface ConnectionSummary {
    id: string;
    timeCreated: number;
    serverId: string;
    sessionId: string;
    state: ConnectionState,
    serverType: TargetType,
    userName: string
}

export interface ListSsmTargetsRequest
{
    showDynamicAccessTargets: boolean;
}

export interface SsmTargetSummary {
    id: string;
    name: string;
    status: SsmTargetStatus;
    environmentId?: string;
    // ID of the agent (hash of public key)
    // Used as the targetId in keysplitting messages
    agentId: string;
    agentVersion: string;
}

export enum AuthenticationType {
    Password = 'Password',
    PrivateKey = 'PrivateKey',
    UseExisting = 'UseExisting'
}

export interface ClusterSummary {
    id: string;
    clusterName: string;
    status: KubeClusterStatus;
    environmentId?: string;
    validUsers: string[];
    agentVersion: string;
    lastAgentUpdate: Date;
}

export interface EnvironmentDetails {
    id: string;
    organizationId: string;
    name: string;
    timeCreated: string;
    resources: EnvironmentResourceDetails[]
    isDefault: boolean;
}

export interface EnvironmentResourceDetails {
    id: string;
    environmentId: string;
    timeCreated: string;
    resourceId: string;
    resourceType: string;
}

export interface ListEnvironmentsRequest {
}

export interface MixpanelTokenResponse
{
    token: string;
}

export interface ClientSecretResponse {
    clientId: string;
    clientSecret: string;
}

export interface MfaTokenRequest
{
    token: string;
}

export interface MfaClearRequest
{
    userId: string;
}

export interface MfaResetRequest
{
    forceSetup: boolean;
}

export interface MfaResetResponse
{
    mfaSecretUrl: string;
}

export interface UserRegisterResponse
{
    userSessionId: string;
    mfaActionRequired: MfaActionRequired;
}

export enum MfaActionRequired
{
    NONE = 'NONE',
    TOTP = 'TOTP',
    RESET ='RESET'
}

export interface UserSummary
{
    id: string;
    organizationId: string;
    fullName: string;
    email: string;
    isAdmin: boolean;
    timeCreated: Date;
}

export interface DynamicAccessConfigSummary
{
    id: string;
    name: string;
    environmentId: string;
}

export interface GetTargetPolicyResponse
{
    allowed: boolean;
    allowedTargetUsers: TargetUser[];
    allowedVerbs: Verb[]
}

export interface TargetUser
{
    userName: string;
}

export interface Verb
{
    type: VerbType;
}

export interface GetTargetPolicyRequest
{
    targetId: string;
    targetType: TargetType;
    verb?: Verb;
    targetUser?: TargetUser;
}

export enum VerbType {
    Shell = 'Shell',
    FileTransfer = 'FileTransfer',
    Tunnel = 'Tunnel'
}

export interface GetAutodiscoveryScriptRequest {
    targetNameScript: string;
    apiUrl: string;
    envId: string;
    agentVersion: string;
}

export interface GetAutodiscoveryScriptResponse {
    autodiscoveryScript: string;
}

export interface ShellConnectionAuthDetails {
    connectionNodeId: string;
    authToken: string;
}
export interface GetKubeUnregisteredAgentYamlRequest {
    clusterName: string;
    labels: string;
    namespace: string;
    environmentId: string;
}
export interface GetKubeUnregisteredAgentYamlResponse {
    yaml: string;
}

export interface KubeProxyRequest {
    clusterName: string;
    clusterUser: string;
    environmentId: string;
}
export interface KubeProxyResponse {
    allowed: boolean;
}

export interface GetAllPoliciesForClusterIdRequest {
    clusterId: string;
}
export interface GetAllPoliciesForClusterIdResponse {
    policies: PolicySummary[]
}

interface PolicySummary {
    name: string;
    id: string;
}

export interface KubernetesPolicySummary {
    id: string;
    name: string;
    metadata: PolicyMetadata
    type: PolicyType
    subjects: Subject[]
    groups: Group[]
    context: KubernetesPolicyContext
}

interface Group {
    id: string;
}

interface KubernetesPolicyContext {
    clusterUsers: { [key: string]: KubernetesPolicyClusterUsers }
    environments: { [key: string]: PolicyEnvironment }
}

export interface KubernetesPolicyClusterUsers {
    name: string;
}

interface PolicyEnvironment {
    id: string;
}

export interface Subject {
    id: string;
    subjectType: SubjectType;
}

interface PolicyMetadata {
    description: string;
}

export enum SubjectType {
    User = 'User',
    ApiKey = 'ApiKey'
}

enum PolicyType {
    TargetConnect = 'TargetConnect',
    OrganizationControls = 'OrganizationControls',
    SessionRecording = 'SessionRecording',
    KubernetesProxy = 'KubernetesProxy'
}

export interface UpdateKubePolicyRequest {
    id: string;
    name: string;
    type: string;
    subjects: Subject[];
    groups: Group[];
    context: string;
    policyMetadata: PolicyMetadata
}

export interface GetUserInfoResponse {
    email: string;
    id: string;
}

export interface GetUserInfoRequest{
    email: string;
}