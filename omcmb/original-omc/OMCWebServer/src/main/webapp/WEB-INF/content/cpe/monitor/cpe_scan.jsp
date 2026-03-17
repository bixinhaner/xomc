<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style>
    .font-size-12::before {
        font-size: 12px;
    }
    .notsupport-pane {
        display: flex;
        align-items: center;
        justify-content: center;
        position: absolute;
        z-index: 10;
        top: 0;
        right: 0;
        bottom: 0;
        left: 0;
        background-color: #fff;
    }
</style>

<div id="cpe_scan_ctn" style="height: 100%;overflow: auto;display: flex;flex-direction: column;background: #fff;border-radius: 5px;">
    <div style="height: 30px;line-height: 30px;padding: 0 20px;">
        <span style="font-size: 14px;font-weight: bold;"><%=rb.getString("ZhanDianSaoMiao")%></span>
        <span style="color: #6969e9;margin-left: 10px;">(<%=rb.getString("ShangCiSaoMiaoShiJian")%> {{lastScanTime}})</span>
        <i @click="clearScan" class="el-icon el-icon-operation-clear" style="position:absolute;top:15px;right:20px;"></i>
    </div>
    <div style="height: 100%;width: 100%;position: relative;overflow: auto;">
        <el-table height="100%" size="mini" stripe border
            :data="list"
        >
            <el-table-column type="index"></el-table-column>
            <el-table-column prop="plmn" label="PLMN"></el-table-column>
            <el-table-column prop="cellId" label="Cell ID"></el-table-column>
            <el-table-column prop="pci" label="PCI"></el-table-column>
            <el-table-column prop="band" label="Band"></el-table-column>
            <el-table-column prop="earfcn" label="EARFCN"></el-table-column>
            <el-table-column prop="bandWidth" label="BW(MHZ)"></el-table-column>
            <el-table-column prop="frequrency" label="Frequency" width="100"></el-table-column>
            <el-table-column prop="rss" label="RSSI(dBm)"></el-table-column>
            <el-table-column prop="rsrp" label="RSRP(dBm)" width="100"></el-table-column>
            <el-table-column prop="rsrq" label="RSRQ(dB)"></el-table-column>
            <el-table-column prop="sinr" label="SINR(dB)"></el-table-column>
        </el-table>
        <div v-if="isNotSupport" class="notsupport-pane">
            <div style="text-align: center;">
                <span class="el-icon el-icon-sas-warning" style="font-size: 45px;color: #7a7d99;"></span>
                <div><%=rb.getString("SheBeiXingHaoBuZhiChi")%></div>
            </div>
        </div>
    </div>
    <div style="padding: 10px 20px;border-top: 1px solid #d5dcec;">
        <el-button v-if="!isScaning" :disabled="isNotSupport" type="primary" @click="startScan">
            <i class="font-size-12 el-icon el-icon-operation-scan" style="margin-right: 5px;"></i> Scan
        </el-button>
        <el-button v-if="isScaning" type="primary" @click="stopScan">
            <i class="font-size-12 el-icon el-icon-operation-terminate" style="margin-right: 5px;"></i> Stop
        </el-button>
        <span class="progress"></span>
        <span v-if="isScaning">
            <div class="progress-cls" style="display: inline-block; margin-right: 5px;"></div>
            <%=rb.getString("SheBeiLiXianTiShi")%>
        </span>
        <span v-if="!isScaning && isFailure">
            <i class="font-size-12 el-icon el-icon-deactivate" style="color: red;"></i>
            <span style="color: red;"><%=rb.getString("XiaoQuSaoMiaoShiBai")%></span>
            <%=rb.getString("XiaoQuSaoMiaoShiBaiYuanYin")%>
        </span>
    </div>
</div>

<script>
new Vue({
    el: '#cpe_scan_ctn',
    data: {
        url: '${ctx}/cpe/cell/scan/queryScanData.action',
        cpeCode: cpeSettingVue.code,
        queryParams: {
            cpeCode: cpeSettingVue.code,
            timeZone: timeZone
        },
        list: [],
        isScaning: false,
        lastScanTime: '',
        isNotSupport: false,
        isFailure: false
    },
    methods: {
        init() {
            var vm = this;

            vm.getScanList();
        },
        timeQuery() {
            var vm = this;

            if(vm.isNotSupport || vm.isScaning == false || !isVisible(document.querySelector('#cpe_scan_ctn'))) {
                return;
            }
            
            vm.getScanList();
        },
        getScanList() {
            var vm = this;

            axios.post(vm.url, stringify(vm.queryParams)).then((res)=>{
                var data = res.data || {};

                if(data.data && !['Send command timeout','NotSupport'].includes(data.data)) {
                    vm.list = [];
                    data.data.split(';').map((item)=> {
                        let arr = item.split(',');

                        vm.list.push({
                            plmn: arr[0],
                            cellId: arr[1],
                            pci: arr[2],
                            band: arr[3],
                            earfcn: arr[4],
                            bandWidth: arr[5],
                            frequrency: arr[6],
                            rss: arr[7],
                            rsrp: arr[8],
                            rsrq: arr[9],
                            sinr: arr[10]
                        });
                    });
                }else if(['Send command timeout','NotSupport'].includes(data.data)) {
                    vm.isNotSupport = ['NotSupport'].includes(data.data);
                    vm.isFailure = ['Send command timeout'].includes(data.data);
                }else {
                    vm.list = [];
                }

                vm.isScaning = ['setStart','setEnd','queryStart'].includes(data.scanState);

                vm.lastScanTime = data.timeStamp;

                if(vm.isNotSupport) {
                    vm.isScaning = false;
                }
            })
        },
        startScan() {
            let vm = this,
                url = '${ctx}/cpe/cell/scan/setScanAbility.action',
                params = {
                    cpeCode: vm.cpeCode
                };

            vm.$confirm('<%=rb.getString("SaoMiaoQueRenTiShi")%>', '<%=rb.getString("QueRen")%>', {
                confirmButtonText: '<%=rb.getString("QueDing")%>',
                cancelButtonText: '<%=rb.getString("QuXiao")%>',
                type: 'warning'
            }).then(()=> {
                vm.isScaning = true;
                axios.post(url, stringify(params)).then((res)=>{
                    let data = res.data || {};
    
                    if(['true', true].includes(data)) {
                        vm.getScanList();
                    }else {
                        vm.$message({
                            message: '<%=rb.getString("ShiBai")%>',
                            type: 'error'
                        })
                    }
                    // vm.isScaning = false;
                })
            }).catch(()=> {
                vm.isScaning = false;
            })
        },
        stopScan() {
            let vm = this,
                url = '${ctx}/cpe/cell/scan/stopScanData.action',
                params = {
                    cpeCode: vm.cpeCode
                };

            axios.post(url, stringify(params)).then((res)=>{
                let data = res.data || {};

                if(['true', true].includes(data)) {
                    vm.getScanList();
                }else {
                    vm.$message({
                        message: '<%=rb.getString("ShiBai")%>',
                        type: 'error'
                    })
                }
            })
        },
        clearScan() {
            let vm = this,
                url = '${ctx}/cpe/cell/scan/clearScanData.action',
                params = {
                    cpeCode: vm.cpeCode
                };

            axios.post(url, stringify(params)).then((res)=>{
                let data = res.data || {};

                if(['true', true].includes(data)) {
                    vm.getScanList();
                }else {
                    vm.$message({
                        message: '<%=rb.getString("ShiBai")%>',
                        type: 'error'
                    })
                }
            })
        }
    },
    mounted() {
        var vm = this;

        vm.init();

        if(window.vm_tb_cpe_cell_scan) {
            clearInterval(window.vm_tb_cpe_cell_scan);
        }

        window.vm_tb_cpe_cell_scan = setInterval(vm.timeQuery, 6000);
    }
})
</script>