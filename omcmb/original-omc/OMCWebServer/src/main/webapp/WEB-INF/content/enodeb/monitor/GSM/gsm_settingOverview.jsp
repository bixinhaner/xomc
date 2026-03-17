<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
#gsmSettingOverviePage {
	background:#f1f1f2;
	height:100%;
	overflow:auto;
	display:flex;
	flex-wrap:wrap;
}
#gsmSettingOverviePage .infoItem{
	border:1px solid #d5dcec;
	border-radius:10px;
	margin:8px;
	background:#fff;
	padding:20px;
	width:100%;
	height:fit-content;
}
#gsmSettingOverviePage .infoTitle {
	height:30px;
	font-size:14px;
	font-weight:bold;
}
#gsmSettingOverviePage .cellItemBoxCls {
	display:flex;
	flex-wrap:wrap;
}
#gsmSettingOverviePage .cellItemBoxCls>div {
	width:24%;
	min-width:200px;
	min-height:55px;
}
#gsmSettingOverviePage .cellItemBoxCls>div:before {
	content:attr(label);
	display:block;
	color:#7a7992;
	margin-bottom:5px;
}
#gsmSettingOverviePage .cellInfoWarp .el-icon-arrow-right:before{
	content:"\e794";
	color:#BBB;
}
</style>
<div class="infoPage" id='gsmSettingOverviePage'>
	<div class="infoItem enbInfo">
		<div class="infoTitle"><%=rb.getString("SASSheBeiXinXi")%></div>
		<div class="cellItemBoxCls">
			<div class="cell-item-cls" label="<%=rb.getString("BSCBianMa")%>">{{rowData.serial_number}}</div>
            <div class="cell-item-cls" label="<%=rb.getString("ChanPinLeiXingBiaoZhi")%>" v-html="productFmt(rowData.product,rowData)"></div>
            <div class="cell-item-cls" label="<%=rb.getString("SheBeiXingHaoMing")%>" v-html="capablityFmt(rowData.module_type,rowData)"></div>
            <div class="cell-item-cls" label="<%=rb.getString("IPDiZhi")%>">{{rowData.cell_ip}}</div>
            <div class="cell-item-cls" label="<%=rb.getString("MACDiZhi")%>">{{rowData.mac_address}}</div>
            <div class="cell-item-cls" label="<%=rb.getString("SoftwareVersion")%>">{{rowData.software_version}}</div>
            <div class="cell-item-cls" label="<%=rb.getString("FirmwareVersion")%>">{{rowData.firmware_version}}</div>
            <div class="cell-item-cls" label="<%=rb.getString("UEShu")%>" v-html="ueCountsFormatter(rowData.ue_count)"></div>
            <div class="cell-item-cls" label="<%=rb.getString("DiYiCiLianJieShiJian")%>">{{rowData.first_online_time}}</div>
			<div class="cell-item-cls" label="<%=rb.getString("ShangCiLianJieShiJian")%>">{{rowData.LASTINFORMTIME}}</div>
			<div class="cell-item-cls" label="<%=rb.getString("YunXingShiJian")%>">{{rowData.up_time}}</div>
			<div class="cell-item-cls" label="<%=rb.getString("LeiJiShiChang")%>">{{rowData.online_duration}}</div>
			<div class="cell-item-cls" label="<%=rb.getString("SheBeiZu")%>">{{rowData.group_name}}</div>
			<div class="cell-item-cls" :label="currentRemarkLabel">{{rowData.remark}}</div>
        </div>
	</div>
</div>
<script type="text/javascript">
var gsmSettingOverviewVue = new Vue({
	el:'#gsmSettingOverviePage',
	data(){
		var vm = this;
		return {
			rowData:{},
            currentRemarkLabel: ''
		}
	},
    computed: {
		gsmRow() { // 当前设备数据
			return gsmvm.selectedRow;
		},
		isSuperAdmin() { //是否是超级管理员
			return is_super_user == 'true';
		},
	},
	watch:{},
	methods:{
		// 初始化
		init(row){
			var vm = this;
            vm.rowData = row;
            vm.getCustomLabelData();
		},
        // 获取自定义label信息
        getCustomLabelData() {
            var vm = this;

            axios.post('${ctx}/cell/columnAlias/queryColumnAliasConfigs.action').then(function(response){
                var data = response.data || [];

                data.forEach(function(item){
                    if(item.columnName == 'remark'){
                        vm.currentRemarkLabel = item.columnAlias || 'Remark';
                    }
                });
            }).catch(function(error){});
        },
        ueCountsFormatter(value, rowData, rowIndex) {
            var vm = this;
            if(value == 0 ){
                return "<a style='color:#000000;text-decoration:none;cursor:default;' href='#'>0</a>"; 
            }else if(value == -1 || value == null){
                return "<a style='color:#000000;text-decoration:none;cursor:default;' href='#'>--</a>"; 
            }else{
                return "<a style='color:#000000;text-decoration:none;' href='#'>"+value+"</a>";
            }
        },
        capablityFmt(value,rowData,rowIndex){
            var rowDatas = rowData,
                capablity = rowDatas.capablity;

            if(capablity == 'enable' && isLWAEnable){
                value = "<span class='cpeLwaKai' style='font-size:22px'>"+ "<span style='font-size:12px;margin-left:25px'>"+(value)+"</span>"+"</span>"
            }else if(capablity =='disable' && isLWAEnable){
                value = "<span class='cpeLwaWu' style='font-size:22px'>"+ "<span style='font-size:12px;margin-left:25px'>"+(value)+"</span>"+"</span>"
            }
            return value;
        },
        productFmt(value, row, index) {
            if ("${carrierType}" == "1") {
                return value;
            }else {
                if(row.have_connected == 2) return '';
                if(value == '--') return '--';

                return value;
            }
        },
	},
	mounted(){
		var vm = this;
        eventBus.$off("gsm-data").$on("gsm-data",this.init)
	}
})
</script> 
