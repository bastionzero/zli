export type OperatingSystem = 'centos' | 'ubuntu' | 'universal'

export type TargetNameManual = {
    name: string
    scheme: 'manual'
}

export type TargetNameDigitalOcean = {
    scheme: 'digitalocean'
}

export type TargetNameAWS = {
    scheme: 'aws'
}

export type TargetNameTime = {
    scheme: 'time'
}

export type TargetNameHostname = {
    scheme: 'hostname'
}

export type TargetName =
    | TargetNameManual
    | TargetNameDigitalOcean
    | TargetNameAWS
    | TargetNameTime
    | TargetNameHostname