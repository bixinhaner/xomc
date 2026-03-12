import{u as j,j as a}from"./index-DvCJCiQT.js";import{r as i}from"./vendor-react-8Uo7Qo2Q.js";import{D as T}from"./index-DAjjw21R.js";import{F as S}from"./index-BHGIeLnr.js";import{L as w}from"./ListPageLayout-BhPPnxeH.js";import{T as N,h as u,B as L,i as B,a4 as I,g as k}from"./vendor-antd-DvN4sx_p.js";const{Text:s,Paragraph:y,Title:g}=N,A=[{id:"1",alarmCode:"ALM-0001",alarmName:"CPU占用率超阈值",severity:"major",neType:"eNB",vendor:"华为",possibleCauses:`1. 系统负载过高
2. 进程异常死循环
3. 内存泄漏导致CPU占用上升`,handlingSuggestions:`1. 检查系统进程，终止异常进程
2. 重启相关服务
3. 如持续发生，考虑升级硬件或优化配置`,updateTime:"2024-02-01"},{id:"2",alarmCode:"ALM-0002",alarmName:"设备断连告警",severity:"critical",neType:"gNB",vendor:"华为",possibleCauses:`1. 网络链路中断
2. 设备断电
3. 配置错误导致连接失败`,handlingSuggestions:`1. 检查网络连接和电源
2. 检查防火墙规则
3. 验证管理IP配置是否正确`,updateTime:"2024-02-05"},{id:"3",alarmCode:"ALM-0003",alarmName:"温度过高告警",severity:"major",neType:"CPE",vendor:"中兴",possibleCauses:`1. 环境温度过高
2. 散热风扇故障
3. 设备长期高负载运行`,handlingSuggestions:`1. 检查机房温度，开启空调降温
2. 检查并更换散热风扇
3. 降低设备工作负载`,updateTime:"2024-02-08"},{id:"4",alarmCode:"ALM-0004",alarmName:"光模块接收功率低",severity:"minor",neType:"eNB",vendor:"爱立信",possibleCauses:`1. 光纤弯折损耗
2. 光模块老化
3. 光纤连接器污染`,handlingSuggestions:`1. 检查光纤路由，避免弯折
2. 清洁光纤连接器
3. 如光模块老化严重，及时更换`,updateTime:"2024-02-10"},{id:"5",alarmCode:"ALM-0005",alarmName:"磁盘空间不足",severity:"warning",neType:"eGW",vendor:"华为",possibleCauses:`1. 日志文件未定期清理
2. 核心转储文件积累
3. 业务数据增长过快`,handlingSuggestions:`1. 定期清理日志文件
2. 配置日志自动归档策略
3. 扩容存储空间`,updateTime:"2024-02-12"},{id:"6",alarmCode:"ALM-0006",alarmName:"链路丢包率异常",severity:"major",neType:"gNB",vendor:"中兴",possibleCauses:`1. 网络拥塞
2. 物理链路质量差
3. QoS配置不当`,handlingSuggestions:`1. 检查网络流量，优化路由
2. 检查物理链路，替换损坏线缆
3. 调整QoS配置`,updateTime:"2024-02-15"},{id:"7",alarmCode:"ALM-0007",alarmName:"软件版本不匹配",severity:"warning",neType:"eNB",vendor:"大唐",possibleCauses:`1. 软件升级后版本兼容性问题
2. 配置文件版本不一致`,handlingSuggestions:`1. 检查软件版本兼容性列表
2. 升级或回退至兼容版本
3. 重新下发配置`,updateTime:"2024-02-18"},{id:"8",alarmCode:"ALM-0008",alarmName:"内存使用率超阈值",severity:"major",neType:"eGW",vendor:"京信",possibleCauses:`1. 内存泄漏
2. 业务并发量过大
3. 缓存配置不合理`,handlingSuggestions:`1. 重启相关服务释放内存
2. 检查内存泄漏点并修复
3. 优化缓存配置`,updateTime:"2024-02-20"}],v={critical:"red",major:"orange",minor:"gold",warning:"blue"};function D(){const e=j(),[n,m]=i.useState({}),[h,d]=i.useState(1),[r,p]=i.useState(null),o=i.useMemo(()=>({critical:e("alarm.severity.critical"),major:e("alarm.severity.major"),minor:e("alarm.severity.minor"),warning:e("alarm.severity.warning")}),[e]),b=i.useMemo(()=>[{name:"alarmCode",label:e("alarm.code"),type:"input"},{name:"alarmName",label:e("alarm.name"),type:"input"},{name:"severity",label:e("alarm.severity"),type:"select",options:[{label:e("alarm.severity.critical"),value:"critical"},{label:e("alarm.severity.major"),value:"major"},{label:e("alarm.severity.minor"),value:"minor"},{label:e("alarm.severity.warning"),value:"warning"}]},{name:"neType",label:e("alarm.neType"),type:"select",options:[{label:"eNB",value:"eNB"},{label:"gNB",value:"gNB"},{label:"CPE",value:"CPE"},{label:"eGW",value:"eGW"}]},{name:"vendor",label:e("device.vendor"),type:"select",options:[{label:"华为",value:"华为"},{label:"中兴",value:"中兴"},{label:"爱立信",value:"爱立信"},{label:"大唐",value:"大唐"},{label:"京信",value:"京信"}]}],[e]),c=i.useMemo(()=>A.filter(l=>!(n.alarmCode&&!l.alarmCode.toLowerCase().includes(String(n.alarmCode).toLowerCase())||n.alarmName&&!l.alarmName.toLowerCase().includes(String(n.alarmName).toLowerCase())||n.severity&&l.severity!==n.severity||n.neType&&l.neType!==n.neType||n.vendor&&l.vendor!==n.vendor)),[n]),x=i.useCallback(l=>{m(l),d(1)},[]),f=i.useCallback(()=>{m({}),d(1)},[]),C=i.useMemo(()=>[{key:"alarmCode",title:e("alarm.code"),dataIndex:"alarmCode",width:110,mono:!0,render:l=>a.jsx(s,{style:{fontFamily:"monospace",fontSize:12,fontWeight:600},children:String(l)})},{key:"alarmName",title:e("alarm.name"),dataIndex:"alarmName",width:180,ellipsis:!0},{key:"severity",title:e("alarm.severity"),dataIndex:"severity",width:80,render:(l,t)=>a.jsx(u,{color:v[t.severity],children:o[t.severity]})},{key:"neType",title:e("alarm.neType"),dataIndex:"neType",width:90},{key:"vendor",title:e("device.vendor"),dataIndex:"vendor",width:90},{key:"possibleCauses",title:e("alarm.content"),dataIndex:"possibleCauses",width:240,ellipsis:!0,render:l=>a.jsxs(s,{type:"secondary",title:String(l),style:{fontSize:12},children:[String(l).split(`
`)[0],"..."]})},{key:"handlingSuggestions",title:e("table.description"),dataIndex:"handlingSuggestions",width:240,ellipsis:!0,render:l=>a.jsxs(s,{type:"secondary",title:String(l),style:{fontSize:12},children:[String(l).split(`
`)[0],"..."]})},{key:"updateTime",title:e("table.updateTime"),dataIndex:"updateTime",width:120},{key:"actions",title:e("table.operation"),dataIndex:"id",width:80,fixed:"right",render:(l,t)=>a.jsx(L,{type:"link",size:"small",icon:a.jsx(B,{}),onClick:()=>p(t),children:e("common.detail")})}],[e,o]);return a.jsxs(a.Fragment,{children:[a.jsxs(w,{title:e("nav.alarm.library"),children:[a.jsx(S,{filterId:"alarm-support-library",fields:b,onSearch:x,onReset:f,collapsedRows:1}),a.jsx(T,{tableId:"alarm-support-library-table",columns:C,dataSource:c,loading:!1,rowKey:"id",total:c.length,pageSize:20,currentPage:h,onPageChange:l=>d(l),defaultDensity:"compact"})]}),a.jsx(I,{title:a.jsxs(k,{children:[a.jsx(s,{style:{fontFamily:"monospace"},children:r?.alarmCode}),a.jsx("span",{children:r?.alarmName}),r&&a.jsx(u,{color:v[r.severity],children:o[r.severity]})]}),open:!!r,onClose:()=>p(null),width:520,children:r&&a.jsxs("div",{style:{display:"flex",flexDirection:"column",gap:20},children:[a.jsxs("div",{children:[a.jsx(s,{type:"secondary",style:{fontSize:12,display:"block",marginBottom:4},children:e("common.detail")}),a.jsx("table",{style:{width:"100%",fontSize:14},children:a.jsx("tbody",{children:[{label:e("alarm.code"),value:r.alarmCode},{label:e("alarm.name"),value:r.alarmName},{label:e("alarm.severity"),value:o[r.severity]},{label:e("alarm.neType"),value:r.neType},{label:e("device.vendor"),value:r.vendor},{label:e("table.updateTime"),value:r.updateTime}].map(({label:l,value:t})=>a.jsxs("tr",{style:{borderBottom:"1px solid #f5f5f5"},children:[a.jsx("td",{style:{padding:"8px 0",color:"#8c8c8c",width:100},children:l}),a.jsx("td",{style:{padding:"8px 0",fontWeight:500},children:t})]},l))})})]}),a.jsxs("div",{children:[a.jsx(g,{level:5,style:{fontSize:14,marginBottom:8,color:"#FA8C16"},children:e("alarm.content")}),a.jsx("div",{style:{background:"#fff7e6",border:"1px solid #ffd591",borderRadius:6,padding:"12px 16px"},children:r.possibleCauses.split(`
`).map((l,t)=>a.jsx(y,{style:{margin:"2px 0",fontSize:13},children:l},t))})]}),a.jsxs("div",{children:[a.jsx(g,{level:5,style:{fontSize:14,marginBottom:8,color:"#52C41A"},children:e("table.description")}),a.jsx("div",{style:{background:"#f6ffed",border:"1px solid #b7eb8f",borderRadius:6,padding:"12px 16px"},children:r.handlingSuggestions.split(`
`).map((l,t)=>a.jsx(y,{style:{margin:"2px 0",fontSize:13},children:l},t))})]})]})})]})}export{D as default};
