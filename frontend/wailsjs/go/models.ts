export namespace types {
	
	export class BoundingBox {
	    index: number;
	    ymin: number;
	    xmin: number;
	    ymax: number;
	    xmax: number;
	
	    static createFrom(source: any = {}) {
	        return new BoundingBox(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.ymin = source["ymin"];
	        this.xmin = source["xmin"];
	        this.ymax = source["ymax"];
	        this.xmax = source["xmax"];
	    }
	}
	export class UsageStats {
	    total_scanned_or_uploaded: number;
	    total_processed: number;
	    total_characters_extracted: number;
	    successful_runs: number;
	    failed_runs: number;
	
	    static createFrom(source: any = {}) {
	        return new UsageStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total_scanned_or_uploaded = source["total_scanned_or_uploaded"];
	        this.total_processed = source["total_processed"];
	        this.total_characters_extracted = source["total_characters_extracted"];
	        this.successful_runs = source["successful_runs"];
	        this.failed_runs = source["failed_runs"];
	    }
	}
	export class Config {
	    session_account: string;
	    api_key: string;
	    base_url: string;
	    model_name: string;
	    auto_extract: boolean;
	    default_output_mode: string;
	    quality: string;
	    releases_repo: string;
	    check_updates_on_startup: boolean;
	    usage_stats: UsageStats;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.session_account = source["session_account"];
	        this.api_key = source["api_key"];
	        this.base_url = source["base_url"];
	        this.model_name = source["model_name"];
	        this.auto_extract = source["auto_extract"];
	        this.default_output_mode = source["default_output_mode"];
	        this.quality = source["quality"];
	        this.releases_repo = source["releases_repo"];
	        this.check_updates_on_startup = source["check_updates_on_startup"];
	        this.usage_stats = this.convertValues(source["usage_stats"], UsageStats);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DocumentPreview {
	    data_url: string;
	    mime_type: string;
	    width: number;
	    height: number;
	
	    static createFrom(source: any = {}) {
	        return new DocumentPreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.data_url = source["data_url"];
	        this.mime_type = source["mime_type"];
	        this.width = source["width"];
	        this.height = source["height"];
	    }
	}
	export class InvoiceValidationResult {
	    matched: boolean;
	    total: number;
	    calculated: number;
	    difference: number;
	    items_count: number;
	
	    static createFrom(source: any = {}) {
	        return new InvoiceValidationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.matched = source["matched"];
	        this.total = source["total"];
	        this.calculated = source["calculated"];
	        this.difference = source["difference"];
	        this.items_count = source["items_count"];
	    }
	}
	export class QueueItem {
	    id: string;
	    file_path: string;
	    file_name: string;
	    file_size_str: string;
	    file_size_bytes: number;
	    status: string;
	    extracted_text: string;
	    error_message: string;
	    source: string;
	    output_mode: string;
	    block_boxes?: BoundingBox[];
	    // Go type: time
	    created_at: any;
	
	    static createFrom(source: any = {}) {
	        return new QueueItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.file_path = source["file_path"];
	        this.file_name = source["file_name"];
	        this.file_size_str = source["file_size_str"];
	        this.file_size_bytes = source["file_size_bytes"];
	        this.status = source["status"];
	        this.extracted_text = source["extracted_text"];
	        this.error_message = source["error_message"];
	        this.source = source["source"];
	        this.output_mode = source["output_mode"];
	        this.block_boxes = this.convertValues(source["block_boxes"], BoundingBox);
	        this.created_at = this.convertValues(source["created_at"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace updater {
	
	export class CheckResult {
	    has_update: boolean;
	    current_version: string;
	    latest_version: string;
	    release_name: string;
	    release_notes: string;
	    release_url: string;
	    published_at: string;
	    download_url: string;
	    asset_name: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new CheckResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.has_update = source["has_update"];
	        this.current_version = source["current_version"];
	        this.latest_version = source["latest_version"];
	        this.release_name = source["release_name"];
	        this.release_notes = source["release_notes"];
	        this.release_url = source["release_url"];
	        this.published_at = source["published_at"];
	        this.download_url = source["download_url"];
	        this.asset_name = source["asset_name"];
	        this.error = source["error"];
	    }
	}

}

export namespace version {
	
	export class Info {
	    version: string;
	    commit: string;
	    build_date: string;
	    go_version: string;
	    os: string;
	    arch: string;
	
	    static createFrom(source: any = {}) {
	        return new Info(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.commit = source["commit"];
	        this.build_date = source["build_date"];
	        this.go_version = source["go_version"];
	        this.os = source["os"];
	        this.arch = source["arch"];
	    }
	}

}

