export interface BoundingBox {
  index: number;
  ymin: number;
  xmin: number;
  ymax: number;
  xmax: number;
}

export interface QueueItem {
  id: string;
  file_path: string;
  file_name: string;
  file_size_str: string;
  file_size_bytes?: number;
  status: 'Ready' | 'Processing' | 'Done' | 'Failed' | string;
  extracted_text: string;
  error_message: string;
  source: 'upload' | 'scanner' | 'clipboard' | string;
  output_mode: 'document' | 'spreadsheet' | 'key_value' | 'raw_text' | string;
  block_boxes?: BoundingBox[];
  created_at?: any;
}

export interface DocumentPreview {
  data_url: string;
  mime_type: string;
  width: number;
  height: number;
}

export interface UsageStats {
  total_scanned_or_uploaded: number;
  total_processed: number;
  total_characters_extracted: number;
  successful_runs: number;
  failed_runs: number;
}

export interface Config {
  session_account: string;
  api_key: string;
  base_url: string;
  model_name: string;
  auto_extract: boolean;
  default_output_mode: 'document' | 'spreadsheet' | 'key_value' | 'raw_text' | string;
  quality: 'standard' | 'high' | string;
  releases_repo: string;
  check_updates_on_startup: boolean;
  usage_stats: UsageStats;
}

export interface InvoiceValidationResult {
  matched: boolean;
  total: number;
  calculated: number;
  difference: number;
  items_count: number;
}

export interface UpdateCheckResult {
  has_update: boolean;
  current_version: string;
  latest_version: string;
  release_name?: string;
  release_notes: string;
  release_url: string;
  published_at?: string;
  download_url: string;
  asset_name?: string;
  error?: string;
}

export interface VersionInfo {
  version: string;
  commit: string;
  build_date: string;
  go_version: string;
  os: string;
  arch: string;
}
