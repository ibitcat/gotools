module Config.Parser {
    let dataMap: {[name: string]: {data: any, fields: string[][]}} = {}

    export function add<T>(data: T, name: string, fields: string[][]) {
        dataMap[name] = {data, fields}
    }

    export function load(binDat: ArrayBuffer) {
        binDat = pako.inflate(binDat)

        let curData: any
        let rowCount: number = null
        let curRowIdx = 0
        let curRow: any
        let curFieldIdx = 0

        let dec = msgpack.Decoder()
        dec.on('data', (v: any) => {
            if (!curData) {
                curData = dataMap[v]
                return
            }
            if (rowCount == null) {
                if (typeof v !== 'number') {
                    console.error('rowCount unpack type error, except number, got', typeof v)
                    return
                }
                if (v === 0) {
                    curData = null
                } else {
                    rowCount = v
                }
                return
            }
            let [field, fieldType] = curData.fields[curFieldIdx]

            if (!curRow) {
                curRow = {}
            }

            curRow[field] = v
            if (fieldType === 'table' || fieldType === 'array' || fieldType === 'object') {
                if (typeof v === 'string') {
                    if (v != "") {
                        try {
                            curRow[field] = JSON.parse(v)
                        } catch (e) {
                            curRow[field] = v
                        }
                    } else {
                        curRow[field] = null
                    }
                }
                if (typeof v === 'object') {
                    fastMod(v)
                }
            }

            ++curFieldIdx
            if (curFieldIdx >= curData.fields.length) {
                fastMod(curRow)
                curData.data[curRow.id] = curRow
                ++curRowIdx
                curRow = null
                curFieldIdx = 0
                if (curRowIdx >= rowCount) {
                    curData = null
                    rowCount = null
                    curRowIdx = 0
                }
            }
            
        }).on('end', () => {
            dataMap = null // 释放没用的数据
        })

        dec.end(binDat)
    }

    function fastMod(v: Object) {
        function tmp() {}
        try { tmp.prototype = v } catch (e) {}
        // try-catch防止v8优化
    }
}
