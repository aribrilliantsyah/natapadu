export namespace models {
	
	export class ActivityLog {
	    id: number;
	    userId: string;
	    username: string;
	    action: string;
	    target: string;
	    details: string;
	    ipAddress: string;
	    // Go type: time
	    createdAt: any;
	
	    static createFrom(source: any = {}) {
	        return new ActivityLog(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.userId = source["userId"];
	        this.username = source["username"];
	        this.action = source["action"];
	        this.target = source["target"];
	        this.details = source["details"];
	        this.ipAddress = source["ipAddress"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
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
	export class ChartPoint {
	    label: string;
	    value: number;
	    secondary: number;
	
	    static createFrom(source: any = {}) {
	        return new ChartPoint(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.value = source["value"];
	        this.secondary = source["secondary"];
	    }
	}
	export class ImportHistory {
	    id: string;
	    templateId: string;
	    templateName?: string;
	    filename: string;
	    fileSizeBytes: number;
	    totalRows: number;
	    successRows: number;
	    failedRows: number;
	    importedBy: string;
	    // Go type: time
	    startedAt: any;
	    // Go type: time
	    finishedAt?: any;
	    status: string;
	    errorMessage?: string;
	
	    static createFrom(source: any = {}) {
	        return new ImportHistory(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.templateId = source["templateId"];
	        this.templateName = source["templateName"];
	        this.filename = source["filename"];
	        this.fileSizeBytes = source["fileSizeBytes"];
	        this.totalRows = source["totalRows"];
	        this.successRows = source["successRows"];
	        this.failedRows = source["failedRows"];
	        this.importedBy = source["importedBy"];
	        this.startedAt = this.convertValues(source["startedAt"], null);
	        this.finishedAt = this.convertValues(source["finishedAt"], null);
	        this.status = source["status"];
	        this.errorMessage = source["errorMessage"];
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
	export class AppSummary {
	    totalTemplates: number;
	    totalRecords: number;
	    totalImports: number;
	    totalUsers: number;
	    databaseSize: string;
	    successRows: number;
	    failedRows: number;
	    recentImports: ImportHistory[];
	    recentActivity: ActivityLog[];
	    importTrend: ChartPoint[];
	    workspaceSizes: ChartPoint[];
	    activityBreakdown: ChartPoint[];
	
	    static createFrom(source: any = {}) {
	        return new AppSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalTemplates = source["totalTemplates"];
	        this.totalRecords = source["totalRecords"];
	        this.totalImports = source["totalImports"];
	        this.totalUsers = source["totalUsers"];
	        this.databaseSize = source["databaseSize"];
	        this.successRows = source["successRows"];
	        this.failedRows = source["failedRows"];
	        this.recentImports = this.convertValues(source["recentImports"], ImportHistory);
	        this.recentActivity = this.convertValues(source["recentActivity"], ActivityLog);
	        this.importTrend = this.convertValues(source["importTrend"], ChartPoint);
	        this.workspaceSizes = this.convertValues(source["workspaceSizes"], ChartPoint);
	        this.activityBreakdown = this.convertValues(source["activityBreakdown"], ChartPoint);
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
	
	export class DistinctValue {
	    value: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new DistinctValue(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = source["value"];
	        this.count = source["count"];
	    }
	}
	export class DuplicateGroup {
	    values: string[];
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new DuplicateGroup(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.values = source["values"];
	        this.count = source["count"];
	    }
	}
	export class ExcelSheetPreview {
	    sheets: string[];
	    activeSheet: string;
	    totalRows: number;
	    totalColumns: number;
	    headerRow: number;
	    dataStartRow: number;
	    headers: string[];
	    sampleRows: string[][];
	
	    static createFrom(source: any = {}) {
	        return new ExcelSheetPreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sheets = source["sheets"];
	        this.activeSheet = source["activeSheet"];
	        this.totalRows = source["totalRows"];
	        this.totalColumns = source["totalColumns"];
	        this.headerRow = source["headerRow"];
	        this.dataStartRow = source["dataStartRow"];
	        this.headers = source["headers"];
	        this.sampleRows = source["sampleRows"];
	    }
	}
	export class FilterCondition {
	    fieldName: string;
	    operator: string;
	    value: any;
	    valueTo?: any;
	
	    static createFrom(source: any = {}) {
	        return new FilterCondition(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fieldName = source["fieldName"];
	        this.operator = source["operator"];
	        this.value = source["value"];
	        this.valueTo = source["valueTo"];
	    }
	}
	export class ExportRequest {
	    templateId: string;
	    format: string;
	    scope: string;
	    selectedRowIds?: number[];
	    columns: string[];
	    searchTerm?: string;
	    filters?: FilterCondition[];
	    sortBy?: string;
	    sortOrder?: string;
	    filterLogic?: string;
	    outputFilename: string;
	
	    static createFrom(source: any = {}) {
	        return new ExportRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.templateId = source["templateId"];
	        this.format = source["format"];
	        this.scope = source["scope"];
	        this.selectedRowIds = source["selectedRowIds"];
	        this.columns = source["columns"];
	        this.searchTerm = source["searchTerm"];
	        this.filters = this.convertValues(source["filters"], FilterCondition);
	        this.sortBy = source["sortBy"];
	        this.sortOrder = source["sortOrder"];
	        this.filterLogic = source["filterLogic"];
	        this.outputFilename = source["outputFilename"];
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
	export class ExportResult {
	    filePath: string;
	    rowCount: number;
	    format: string;
	
	    static createFrom(source: any = {}) {
	        return new ExportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.filePath = source["filePath"];
	        this.rowCount = source["rowCount"];
	        this.format = source["format"];
	    }
	}
	
	export class ImportError {
	    id: number;
	    importId: string;
	    rowNumber: number;
	    columnName: string;
	    fieldValue: string;
	    errorReason: string;
	    // Go type: time
	    createdAt: any;
	
	    static createFrom(source: any = {}) {
	        return new ImportError(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.importId = source["importId"];
	        this.rowNumber = source["rowNumber"];
	        this.columnName = source["columnName"];
	        this.fieldValue = source["fieldValue"];
	        this.errorReason = source["errorReason"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
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
	
	export class QueryRequest {
	    templateId: string;
	    page: number;
	    pageSize: number;
	    searchTerm: string;
	    sortBy: string;
	    sortOrder: string;
	    filters: FilterCondition[];
	    filterLogic: string;
	
	    static createFrom(source: any = {}) {
	        return new QueryRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.templateId = source["templateId"];
	        this.page = source["page"];
	        this.pageSize = source["pageSize"];
	        this.searchTerm = source["searchTerm"];
	        this.sortBy = source["sortBy"];
	        this.sortOrder = source["sortOrder"];
	        this.filters = this.convertValues(source["filters"], FilterCondition);
	        this.filterLogic = source["filterLogic"];
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
	export class TemplateColumn {
	    id: string;
	    templateId: string;
	    excelColumn: string;
	    fieldName: string;
	    displayName: string;
	    dataType: string;
	    formatPattern: string;
	    required: boolean;
	    isUnique: boolean;
	    defaultValue: string;
	    transformRules: string;
	    validationRules: string;
	    sortOrder: number;
	    isIndexed: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TemplateColumn(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.templateId = source["templateId"];
	        this.excelColumn = source["excelColumn"];
	        this.fieldName = source["fieldName"];
	        this.displayName = source["displayName"];
	        this.dataType = source["dataType"];
	        this.formatPattern = source["formatPattern"];
	        this.required = source["required"];
	        this.isUnique = source["isUnique"];
	        this.defaultValue = source["defaultValue"];
	        this.transformRules = source["transformRules"];
	        this.validationRules = source["validationRules"];
	        this.sortOrder = source["sortOrder"];
	        this.isIndexed = source["isIndexed"];
	    }
	}
	export class QueryResponse {
	    data: any[];
	    totalRows: number;
	    page: number;
	    pageSize: number;
	    totalPages: number;
	    columns: TemplateColumn[];
	    executionMs: number;
	
	    static createFrom(source: any = {}) {
	        return new QueryResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.data = source["data"];
	        this.totalRows = source["totalRows"];
	        this.page = source["page"];
	        this.pageSize = source["pageSize"];
	        this.totalPages = source["totalPages"];
	        this.columns = this.convertValues(source["columns"], TemplateColumn);
	        this.executionMs = source["executionMs"];
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
	export class SavedFilter {
	    id: string;
	    templateId: string;
	    name: string;
	    filterPayload: string;
	    createdBy: string;
	    // Go type: time
	    createdAt: any;
	
	    static createFrom(source: any = {}) {
	        return new SavedFilter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.templateId = source["templateId"];
	        this.name = source["name"];
	        this.filterPayload = source["filterPayload"];
	        this.createdBy = source["createdBy"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
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
	export class Template {
	    id: string;
	    name: string;
	    description: string;
	    sheetName: string;
	    headerRow: number;
	    dataStartRow: number;
	    version: number;
	    status: string;
	    columns: TemplateColumn[];
	    recordCount: number;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Template(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.sheetName = source["sheetName"];
	        this.headerRow = source["headerRow"];
	        this.dataStartRow = source["dataStartRow"];
	        this.version = source["version"];
	        this.status = source["status"];
	        this.columns = this.convertValues(source["columns"], TemplateColumn);
	        this.recordCount = source["recordCount"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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
	
	export class User {
	    id: string;
	    username: string;
	    displayName: string;
	    role: string;
	    status: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new User(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.username = source["username"];
	        this.displayName = source["displayName"];
	        this.role = source["role"];
	        this.status = source["status"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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
	export class UserSession {
	    user: User;
	    token: string;
	    // Go type: time
	    expiresAt: any;
	
	    static createFrom(source: any = {}) {
	        return new UserSession(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.user = this.convertValues(source["user"], User);
	        this.token = source["token"];
	        this.expiresAt = this.convertValues(source["expiresAt"], null);
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

