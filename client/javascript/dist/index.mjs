import axios from 'axios';

/*! *****************************************************************************
Copyright (c) Microsoft Corporation. All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License"); you may not use
this file except in compliance with the License. You may obtain a copy of the
License at http://www.apache.org/licenses/LICENSE-2.0

THIS CODE IS PROVIDED ON AN *AS IS* BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
KIND, EITHER EXPRESS OR IMPLIED, INCLUDING WITHOUT LIMITATION ANY IMPLIED
WARRANTIES OR CONDITIONS OF TITLE, FITNESS FOR A PARTICULAR PURPOSE,
MERCHANTABLITY OR NON-INFRINGEMENT.

See the Apache Version 2.0 License for specific language governing permissions
and limitations under the License.
***************************************************************************** */
/* global Reflect, Promise */

var extendStatics = function(d, b) {
    extendStatics = Object.setPrototypeOf ||
        ({ __proto__: [] } instanceof Array && function (d, b) { d.__proto__ = b; }) ||
        function (d, b) { for (var p in b) if (b.hasOwnProperty(p)) d[p] = b[p]; };
    return extendStatics(d, b);
};

function __extends(d, b) {
    extendStatics(d, b);
    function __() { this.constructor = d; }
    d.prototype = b === null ? Object.create(b) : (__.prototype = b.prototype, new __());
}

function __awaiter(thisArg, _arguments, P, generator) {
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : new P(function (resolve) { resolve(result.value); }).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
}

function __generator(thisArg, body) {
    var _ = { label: 0, sent: function() { if (t[0] & 1) throw t[1]; return t[1]; }, trys: [], ops: [] }, f, y, t, g;
    return g = { next: verb(0), "throw": verb(1), "return": verb(2) }, typeof Symbol === "function" && (g[Symbol.iterator] = function() { return this; }), g;
    function verb(n) { return function (v) { return step([n, v]); }; }
    function step(op) {
        if (f) throw new TypeError("Generator is already executing.");
        while (_) try {
            if (f = 1, y && (t = op[0] & 2 ? y["return"] : op[0] ? y["throw"] || ((t = y["return"]) && t.call(y), 0) : y.next) && !(t = t.call(y, op[1])).done) return t;
            if (y = 0, t) op = [op[0] & 2, t.value];
            switch (op[0]) {
                case 0: case 1: t = op; break;
                case 4: _.label++; return { value: op[1], done: false };
                case 5: _.label++; y = op[1]; op = [0]; continue;
                case 7: op = _.ops.pop(); _.trys.pop(); continue;
                default:
                    if (!(t = _.trys, t = t.length > 0 && t[t.length - 1]) && (op[0] === 6 || op[0] === 2)) { _ = 0; continue; }
                    if (op[0] === 3 && (!t || (op[1] > t[0] && op[1] < t[3]))) { _.label = op[1]; break; }
                    if (op[0] === 6 && _.label < t[1]) { _.label = t[1]; t = op; break; }
                    if (t && _.label < t[2]) { _.label = t[2]; _.ops.push(op); break; }
                    if (t[2]) _.ops.pop();
                    _.trys.pop(); continue;
            }
            op = body.call(thisArg, _);
        } catch (e) { op = [6, e]; y = 0; } finally { f = t = 0; }
        if (op[0] & 5) throw op[1]; return { value: op[0] ? op[1] : void 0, done: true };
    }
}

var Record = /** @class */ (function () {
    function Record(name, ttl, type) {
        if (ttl === void 0) { ttl = 3600; }
        if (type === void 0) { type = 'UNKNOWN'; }
        this.name = name;
        this.ttl = ttl;
        this.type = type;
    }
    Record.parse = function (str) {
        var m = str.match(/[^ \t]+[ \t]+[0-9]+[ \t]+IN[ \t]+([A-Z]+)[ \t]+.*/);
        if (m === null) {
            return null;
        }
        var func = {
            A: ARecord.parse,
            AAAA: AaaaRecord.parse,
            CNAME: CnameRecord.parse,
            PTR: PtrRecord.parse,
            TXT: TxtRecord.parse,
            SRV: SrvRecord.parse,
        }[m[1]];
        if (func !== null) {
            return func(str);
        }
        return null;
    };
    return Record;
}());
var ARecord = /** @class */ (function (_super) {
    __extends(ARecord, _super);
    function ARecord(name, address, ttl) {
        if (ttl === void 0) { ttl = 3600; }
        var _this = _super.call(this, name, ttl, 'A') || this;
        _this.address = address;
        return _this;
    }
    ARecord.parse = function (str) {
        var m = str.match(/([^ \t]+)[ \t]+([0-9]+)[ \t]+IN[ \t]+A[ \t]+([^ \t]+)/);
        if (m !== null) {
            return new ARecord(m[1], m[3], parseInt(m[2]));
        }
        return null;
    };
    ARecord.prototype.toString = function () {
        return this.name + " " + this.ttl + " IN A " + this.address;
    };
    return ARecord;
}(Record));
var AaaaRecord = /** @class */ (function (_super) {
    __extends(AaaaRecord, _super);
    function AaaaRecord(name, address, ttl) {
        if (ttl === void 0) { ttl = 3600; }
        var _this = _super.call(this, name, ttl, 'AAAA') || this;
        _this.address = address;
        return _this;
    }
    AaaaRecord.parse = function (str) {
        var m = str.match(/([^ \t]+)[ \t]+([0-9]+)[ \t]+IN[ \t]+AAAA[ \t]+([^ \t]+)/);
        if (m !== null) {
            return new AaaaRecord(m[1], m[3], parseInt(m[2]));
        }
        return null;
    };
    AaaaRecord.prototype.toString = function () {
        return this.name + " " + this.ttl + " IN AAAA " + this.address;
    };
    return AaaaRecord;
}(Record));
var CnameRecord = /** @class */ (function (_super) {
    __extends(CnameRecord, _super);
    function CnameRecord(name, target, ttl) {
        if (ttl === void 0) { ttl = 3600; }
        var _this = _super.call(this, name, ttl, 'CNAME') || this;
        _this.target = target;
        return _this;
    }
    CnameRecord.parse = function (str) {
        var m = str.match(/([^ \t]+)[ \t]+([0-9]+)[ \t]+IN[ \t]+CNAME[ \t]+([^ \t]+)/);
        if (m !== null) {
            return new CnameRecord(m[1], m[3], parseInt(m[2]));
        }
        return null;
    };
    CnameRecord.prototype.toString = function () {
        return this.name + " " + this.ttl + " IN CNAME " + this.target;
    };
    return CnameRecord;
}(Record));
var PtrRecord = /** @class */ (function (_super) {
    __extends(PtrRecord, _super);
    function PtrRecord(name, domain, ttl) {
        if (ttl === void 0) { ttl = 3600; }
        var _this = _super.call(this, name, ttl, 'PTR') || this;
        _this.domain = domain;
        return _this;
    }
    PtrRecord.parse = function (str) {
        var m = str.match(/([^ \t]+)[ \t]+([0-9]+)[ \t]+IN[ \t]+PTR[ \t]+([^ \t]+)/);
        if (m !== null) {
            return new PtrRecord(m[1], m[3], parseInt(m[2]));
        }
        return null;
    };
    PtrRecord.prototype.toString = function () {
        return this.name + " " + this.ttl + " IN PTR " + this.domain;
    };
    return PtrRecord;
}(Record));
var TxtRecord = /** @class */ (function (_super) {
    __extends(TxtRecord, _super);
    function TxtRecord(name, text, ttl) {
        if (ttl === void 0) { ttl = 3600; }
        var _this = _super.call(this, name, ttl, 'TXT') || this;
        _this.text = text;
        return _this;
    }
    TxtRecord.parse = function (str) {
        var m = str.match(/([^ \t]+)[ \t]+([0-9]+)[ \t]+IN[ \t]+TXT[ \t]+("[^"]*")+/);
        if (m !== null) {
            return new TxtRecord(m[1], m[3], parseInt(m[2]));
        }
        return null;
    };
    TxtRecord.prototype.toString = function () {
        return this.name + " " + this.ttl + " IN TXT \"" + this.text + "\"";
    };
    return TxtRecord;
}(Record));
var SrvRecord = /** @class */ (function (_super) {
    __extends(SrvRecord, _super);
    function SrvRecord(name, target, port, priority, weight, ttl) {
        if (priority === void 0) { priority = 0; }
        if (weight === void 0) { weight = 0; }
        if (ttl === void 0) { ttl = 3600; }
        var _this = _super.call(this, name, ttl, 'SRV') || this;
        _this.target = target;
        _this.port = port;
        _this.priority = priority;
        _this.weight = weight;
        return _this;
    }
    SrvRecord.parse = function (str) {
        var m = str.match(/([^ \t]+)[ \t]+([0-9]+)[ \t]+IN[ \t]+SRV[ \t]+([0-9]+)[ \t]+([0-9]+)[ \t]+([0-9]+)[ \t]+("[^"]*")+/);
        if (m !== null) {
            return new SrvRecord(m[1], m[6], parseInt(m[5]), parseInt(m[3]), parseInt(m[4]), parseInt(m[2]));
        }
        return null;
    };
    SrvRecord.prototype.toString = function () {
        return this.name + " " + this.ttl + " IN SRV " + this.priority + " " + this.weight + " " + this.port + " " + this.target;
    };
    return SrvRecord;
}(Record));
function parseRecords(text) {
    return text.split('\n').map(function (line) { return Record.parse(line); }).filter(function (record) { return record !== null; });
}
var Landns = /** @class */ (function () {
    function Landns(endpoint) {
        if (endpoint === void 0) { endpoint = 'http://localhost:9353/api/v1'; }
        this.endpoint = endpoint;
    }
    Landns.prototype.set = function (records) {
        return __awaiter(this, void 0, void 0, function () {
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0: return [4 /*yield*/, axios.post(this.endpoint, records.filter(function (record) { return record !== null; }).map(function (record) { return record.toString(); }).join('\n'), { headers: { "Content-Type": "text/plain" } })];
                    case 1:
                        _a.sent();
                        return [2 /*return*/];
                }
            });
        });
    };
    Landns.prototype.remove = function (id) {
        return __awaiter(this, void 0, void 0, function () {
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0: return [4 /*yield*/, axios["delete"](this.endpoint + "/id/" + id)];
                    case 1:
                        _a.sent();
                        return [2 /*return*/];
                }
            });
        });
    };
    Landns.prototype.get = function () {
        return __awaiter(this, void 0, void 0, function () {
            var resp;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0: return [4 /*yield*/, axios.get(this.endpoint)];
                    case 1:
                        resp = _a.sent();
                        return [2 /*return*/, parseRecords(resp.data)];
                }
            });
        });
    };
    Landns.prototype.glob = function (query) {
        return __awaiter(this, void 0, void 0, function () {
            var resp;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0: return [4 /*yield*/, axios.get(this.endpoint + "/glob/" + query)];
                    case 1:
                        resp = _a.sent();
                        return [2 /*return*/, parseRecords(resp.data)];
                }
            });
        });
    };
    return Landns;
}());

export { ARecord, AaaaRecord, CnameRecord, Landns, PtrRecord, Record, SrvRecord, TxtRecord, parseRecords };
