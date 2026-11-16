import type { components, operations } from './openapi';

export type { components, operations, paths } from './openapi';

type Schemas = components['schemas'];

export type Tunnel = Schemas['TunnelState'];
export type CreateTunnelInput = Required<Pick<Schemas['TunnelBody'], 'name' | 'provider' | 'localPort'>>;
export type UpdateTunnelInput = Pick<Schemas['TunnelBody'], 'name' | 'provider' | 'localPort'>;
export type Provider = Schemas['ProviderInfo'];
export type ProviderBinaryRow = Schemas['ProviderBinaryRow'];
export type Settings = Schemas['Settings'];
export type SettingsPatch = Pick<
  Schemas['SettingsPatch'],
  'autostart_enabled' | 'language' | 'notification_sound' | 'notifications_enabled'
>;
export type UpdateInfo = Schemas['UpdateInfo'];
export type UpdateMethod = UpdateInfo['method'];
export type StatusResponse = Schemas['StatusOutputBody'];
export type LogSnapshot = Schemas['LogSnapshot'];

type StreamOf<Op extends keyof operations> =
  operations[Op]['responses'] extends { 200: { content: { 'text/event-stream': infer E } } } ? E : never;

/** One event of `GET /api/events`, by name. */
export type EventFrame = StreamOf<'events'>;
/** One event of `GET /api/tunnels/{id}/logs?follow=1`, by name. */
export type LogFrame = StreamOf<'tunnel-logs'>;

export type EventName = EventFrame['event'];
export type EventData<N extends EventName> = Extract<EventFrame, { event: N }>['data'];
export type LogEventName = LogFrame['event'];
