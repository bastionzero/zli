export const ShellHubIncomingMessages = {
    shellOutput: "ShellOutput",
    shellDisconnect: "ShellDisconnect",
    shellStart: "ShellStart"
}
    
export const ShellHubOutgoingMessages = {
    shellConnect: "ShellConnect",
    shellInput: "ShellInput",
    shellGeometry: "ShellGeometry"
}

export interface ShellState {
    loading: boolean;
    disconnected: boolean;
}