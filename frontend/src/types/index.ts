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
  status: 'Ready' | 'Processing' | 'Done' | 'Failed';
  extracted_text: string;
  error_message: string;
  source: 'upload' | 'scanner' | 'clipboard';
  output_mode: 'document' | 'spreadsheet' | 'key_value' | 'raw_text';
  block_boxes?: BoundingBox[];
  created_at?: string;
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
  default_output_mode: 'document' | 'spreadsheet' | 'key_value' | 'raw_text';
  quality: 'standard' | 'high';
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
  release_notes: string;
  release_url: string;
  download_url: string;
  error?: string;
}
