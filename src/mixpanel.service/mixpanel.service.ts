import { TargetType } from '../types';
import { Dictionary } from 'lodash';
import mixpanel, { Mixpanel, track } from 'mixpanel';
import { TrackNewConnection } from './mixpanel.service.types';


export class MixpanelService
{
    private mixpanelClient: Mixpanel;
    private userId: string;

    constructor(mixpanelToken: string, userId: string)
    {
        this.mixpanelClient = mixpanel.init(mixpanelToken, {
            protocol: 'https',
        });

        this.userId = userId;
    }


    // track connect calls
    public TrackNewConnection(targetType: TargetType): void
    {
        const trackMessage : TrackNewConnection = {
            distinct_id: this.userId,
            client_type: "CLI",
            ConnectionType: targetType
        };

        this.mixpanelClient.track('ConnectionOpened', trackMessage);
    }

    public TrackCliCall(eventName: string, properties: Dictionary<string | string[] | unknown>)
    {
        properties.distinct_id = this.userId;
        properties.client_type = "CLI";

        this.mixpanelClient.track(eventName, properties);
    }
}