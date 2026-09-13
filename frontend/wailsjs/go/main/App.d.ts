import { Config, QueueItem, UsageStats, BoundingBox, InvoiceValidationResult, UpdateCheckResult } from '../../../src/types';

export function GetConfig(): Promise<Config>;
export function SaveConfig(arg1: Config): Promise<void>;
export function GetUsageStats(): Promise<UsageStats>;
export function LoadHistory(): Promise<QueueItem[]>;
export function SaveHistory(arg1: QueueItem[]): Promise<void>;
export function PickFiles(): Promise<QueueItem[]>;
export function GetClipboardImage(): Promise<QueueItem>;
export function ScanDocument(): Promise<QueueItem>;
export function ExtractText(filePath: string, mode: string, quality: string): Promise<string>;
export function AlignBlocks(filePath: string, blocks: string[]): Promise<BoundingBox[]>;
export function TransformText(text: string, transformType: string): Promise<string>;
export function CheckInvoiceMath(text: string): Promise<InvoiceValidationResult | null>;
export function SaveExportFile(defaultFilename: string, content: string): Promise<string>;
export function CheckForUpdates(): Promise<UpdateCheckResult>;
export function GenerateBugReport(description: string, steps: string): Promise<string>;
