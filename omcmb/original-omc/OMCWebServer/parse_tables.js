const fs = require('fs');

const files = {
    'enodeb': 'src/main/webapp/WEB-INF/content/enodeb/monitor/enodeb_monitor_vue.jsp',
    'gsm': 'src/main/webapp/WEB-INF/content/enodeb/monitor/GSM/gsm_monitor_vue.jsp',
    'gnodeb': 'src/main/webapp/WEB-INF/content/gnodeb/monitor/gnodeb_monitor.jsp',
    'cpe': 'src/main/webapp/WEB-INF/content/cpe/monitor/cpe_monitor_vue.jsp'
};

const fileFields = {};

for (let name in files) {
    try {
        const content = fs.readFileSync(files[name], 'utf-8');
        const fields = new Set();
        
        // This regex matches any \<el-table-column ... prop="value" ... >\
        // Allows multiline attributes
        const columnRegex = /<el-table-column[^>]*?prop=["']([^"']+)["'][^>]*>/gi;
        
        let match;
        while ((match = columnRegex.exec(content)) !== null) {
            fields.add(match[1]);
        }
        
        fileFields[name] = Array.from(fields);
        console.log(\Found \ fields in \\);
    } catch (e) {
        console.error(\Error reading \: \\);
    }
}

const normalizedMap = {};
const originalCasing = {};

for (let name in fileFields) {
    normalizedMap[name] = fileFields[name].map(f => f.toLowerCase());
    fileFields[name].forEach(f => {
        if (!originalCasing[f.toLowerCase()]) {
            originalCasing[f.toLowerCase()] = f;
        }
    });
}

const allFieldsLower = new Set();
for (let name in normalizedMap) {
    normalizedMap[name].forEach(f => allFieldsLower.add(f));
}

const publicFields = [];
const uniqueFields = {};
Object.keys(files).forEach(name => uniqueFields[name] = []);
const partialFields = [];

allFieldsLower.forEach(f => {
    const presence = [];
    for (let name in normalizedMap) {
        if (normalizedMap[name].includes(f)) {
            presence.push(name);
        }
    }
    
    if (presence.length === Object.keys(files).length) {
        publicFields.push(f);
    } else if (presence.length === 1) {
        uniqueFields[presence[0]].push(f);
    } else {
        partialFields.push({ field: f, presence });
    }
});

let mdContent = \# 监控页面列表字段全面对比分析

此文档是对四个核心监控页面表格（\\\<el-table-column>\\\）中所使用的字段（\\\prop\\\）的详尽对比。

## 分析的页面：
1. \\\enodeb_monitor_vue.jsp\\\
2. \\\gsm_monitor_vue.jsp\\\
3. \\\gnodeb_monitor.jsp\\\
4. \\\cpe_monitor_vue.jsp\\\

---

## 1. 公共字段 (4个页面共有)
这些字段在所有四个页面的表格中均有定义：
\;

if (publicFields.length > 0) {
    publicFields.sort().forEach(f => {
        mdContent += \- \\\\\\\\\n\;
    });
} else {
    mdContent += \*无*\\n\;
}

mdContent += \\\n## 2. 独有字段\\n这些字段仅在特定的页面中出现：\\n\;
for (let name in uniqueFields) {
    mdContent += \### [\] 独有\\n\;
    if (uniqueFields[name].length > 0) {
        uniqueFields[name].sort().forEach(f => {
            mdContent += \- \\\\\\\\\n\;
        });
    } else {
        mdContent += \*无*\\n\;
    }
}

mdContent += \\\n## 3. 部分页面共有字段\\n这些字段存在于2到3个页面中：\\n\;
partialFields.sort((a, b) => originalCasing[a.field].localeCompare(originalCasing[b.field])).forEach(pf => {
    mdContent += \- \\\\\\\: 存在于 \\\n\;
});

fs.writeFileSync('monitor_fields_comparison.md', mdContent, 'utf-8');
console.log('Markdown generated successfully!');
