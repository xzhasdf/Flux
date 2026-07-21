export namespace main {
	
	export class ConvertResult {
	    src: string;
	    dst: string;
	    success: boolean;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new ConvertResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.src = source["src"];
	        this.dst = source["dst"];
	        this.success = source["success"];
	        this.message = source["message"];
	    }
	}
	export class DownloadRequest {
	    urls: string[];
	    saveDir: string;
	    quality: string;
	    format: string;
	    subtitle: boolean;
	    embedSub: boolean;
	    thumbnail: boolean;
	    danmaku: boolean;
	    cookieMode: string;
	    cookieFile: string;
	    proxy: string;
	    rateLimit: string;
	    playlistItems: string;
	    workers: number;
	    fragThreads: number;
	
	    static createFrom(source: any = {}) {
	        return new DownloadRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.urls = source["urls"];
	        this.saveDir = source["saveDir"];
	        this.quality = source["quality"];
	        this.format = source["format"];
	        this.subtitle = source["subtitle"];
	        this.embedSub = source["embedSub"];
	        this.thumbnail = source["thumbnail"];
	        this.danmaku = source["danmaku"];
	        this.cookieMode = source["cookieMode"];
	        this.cookieFile = source["cookieFile"];
	        this.proxy = source["proxy"];
	        this.rateLimit = source["rateLimit"];
	        this.playlistItems = source["playlistItems"];
	        this.workers = source["workers"];
	        this.fragThreads = source["fragThreads"];
	    }
	}
	export class EpisodeInfo {
	    title: string;
	    duration: number;
	
	    static createFrom(source: any = {}) {
	        return new EpisodeInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.duration = source["duration"];
	    }
	}
	export class HistoryRecord {
	    id: string;
	    type: string;
	    title: string;
	    count: number;
	    saveDir: string;
	    status: string;
	    detail: string;
	    payload: string;
	    createdAt: number;
	    updatedAt: number;
	
	    static createFrom(source: any = {}) {
	        return new HistoryRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.title = source["title"];
	        this.count = source["count"];
	        this.saveDir = source["saveDir"];
	        this.status = source["status"];
	        this.detail = source["detail"];
	        this.payload = source["payload"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class JobInfo {
	    id: string;
	    seq: number;
	    title: string;
	
	    static createFrom(source: any = {}) {
	        return new JobInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.seq = source["seq"];
	        this.title = source["title"];
	    }
	}
	export class M3U8Request {
	    urls: string[];
	    saveDir: string;
	
	    static createFrom(source: any = {}) {
	        return new M3U8Request(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.urls = source["urls"];
	        this.saveDir = source["saveDir"];
	    }
	}
	export class MagnetRequest {
	    links: string[];
	    saveDir: string;
	    dlLimit: string;
	    ulLimit: string;
	    maxConn: number;
	    seedTime: string;
	    extraTracker: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MagnetRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.links = source["links"];
	        this.saveDir = source["saveDir"];
	        this.dlLimit = source["dlLimit"];
	        this.ulLimit = source["ulLimit"];
	        this.maxConn = source["maxConn"];
	        this.seedTime = source["seedTime"];
	        this.extraTracker = source["extraTracker"];
	    }
	}
	export class NCMConvertRequest {
	    inputDir: string;
	    outputDir: string;
	    format: string;
	    bitrate: string;
	    overwrite: boolean;
	    workers: number;
	    extensions: string[];
	
	    static createFrom(source: any = {}) {
	        return new NCMConvertRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.inputDir = source["inputDir"];
	        this.outputDir = source["outputDir"];
	        this.format = source["format"];
	        this.bitrate = source["bitrate"];
	        this.overwrite = source["overwrite"];
	        this.workers = source["workers"];
	        this.extensions = source["extensions"];
	    }
	}
	export class NCMConvertResponse {
	    total: number;
	    success: number;
	    failed: number;
	    results: ConvertResult[];
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new NCMConvertResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.success = source["success"];
	        this.failed = source["failed"];
	        this.results = this.convertValues(source["results"], ConvertResult);
	        this.error = source["error"];
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
	export class ParseInfoRequest {
	    url: string;
	    cookieMode: string;
	    cookieFile: string;
	    proxy: string;
	
	    static createFrom(source: any = {}) {
	        return new ParseInfoRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.cookieMode = source["cookieMode"];
	        this.cookieFile = source["cookieFile"];
	        this.proxy = source["proxy"];
	    }
	}
	export class VideoInfo {
	    title: string;
	    uploader: string;
	    duration: number;
	    thumbnail: string;
	    description: string;
	    uploadDate: string;
	    viewCount: number;
	    resolutions: number[];
	    filesize: number;
	    isPlaylist: boolean;
	    playlistTitle?: string;
	    episodes?: EpisodeInfo[];
	
	    static createFrom(source: any = {}) {
	        return new VideoInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.uploader = source["uploader"];
	        this.duration = source["duration"];
	        this.thumbnail = source["thumbnail"];
	        this.description = source["description"];
	        this.uploadDate = source["uploadDate"];
	        this.viewCount = source["viewCount"];
	        this.resolutions = source["resolutions"];
	        this.filesize = source["filesize"];
	        this.isPlaylist = source["isPlaylist"];
	        this.playlistTitle = source["playlistTitle"];
	        this.episodes = this.convertValues(source["episodes"], EpisodeInfo);
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

