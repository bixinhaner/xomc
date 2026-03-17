<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
.infoPage {
	background:#f1f1f2;
	height:100%;
	overflow:auto;
	display:flex;
	flex-wrap:wrap;
}
.infoItem{
	border:1px solid #d5dcec;
	border-radius:10px;
	margin:8px;
	background:#fff;
	padding:20px;
	width:100%;
	height:fit-content;
}
.infoItem.alarmContent {
	display:flex;
}
.alarmItem{
	border-right:1px solid #d5dcec;
	flex:20%;
	text-align:center;
}
.infoTitle {
	height:30px;
	font-size:14px;
	font-weight:bold;
}
.cellItemBoxCls {
	display:flex;
	flex-wrap:wrap;
}
.cellItemBoxCls>div {
	width:20%;
	min-width:200px;
	min-height:55px;
}
.cellItemBoxCls>div:before {
	content:attr(label);
	display:block;
	color:#7a7992;
	margin-bottom:5px;
}
.infoItem.halfItem{
	width:45%;
	min-width:400px;
	flex:1;
}
.infoItem.halfItem .cellItemBoxCls>div {
	width:30%;
	min-width:120px;
}
</style>
<div class="infoPage" id='cpeDetailPage'>
	<div class="infoItem enbInfo">
		<div class="infoTitle"><%=rb.getString("SASSheBeiXinXi")%></div>
		<div class="cellItemBoxCls">
			<div class="cell-item-cls vertic" label="<%=rb.getString("CPEXuLieHao")%>">${serialNumber}</div>
			<div class="cell-item-cls vertic" label="<%=rb.getString("CPEName")%>">${cpeName}</div>
			<div class="cell-item-cls vertic" label="<%=rb.getString("ChanPinXingHao")%>" v-html="modelName"></div>
			<div class="cell-item-cls vertic" label="<%=rb.getString("SoftwareVersion")%>">${softwareVersion}</div>
			<div class="cell-item-cls vertic" label="<%=rb.getString("MoKuaiMingCheng")%>">${moduleName}</div>
			
			<div class="cell-item-cls vertic" label="<%=rb.getString("DiYiCiLianJieShiJian")%>">${firstPeriodTime}</div>
			<div class="cell-item-cls vertic" label="<%=rb.getString("ShangCiLianJieShiJian")%>">${lastPeriodTime}</div>
			<div class="cell-item-cls vertic" label="<%=rb.getString("YunXingShiJian")%>">${upTime}</div>
			<div class="cell-item-cls vertic" label="MCC">${MCC}</div>
			<div class="cell-item-cls vertic" label="MNC">${MNC}</div>
			
			<div class="cell-item-cls vertic" label="Module" v-html="moduleFmt"></div>
			<div class="cell-item-cls vertic" label="<%=rb.getString("SheBeiZu")%>">${groupName}</div>
			
			
		</div>
	</div>
	<div class="infoItem enbInfo" v-if="!wanShow && !isR005 && !is43XAP">
		<div class="infoTitle">LTE <%=rb.getString("ZhuangTai")%></div>
		<div class="cellItemBoxCls">
			<div class="cell-item-cls vertic" label="<%=rb.getString("IMSI")%>">${IMSI}</div>
			<div class="cell-item-cls vertic" label="PCI">${PCI}</div>
			<div class="cell-item-cls vertic" label="ECI">${ECI}</div>
			<div class="cell-item-cls vertic" label="<%=rb.getString("PinDian")%>" v-html="earfcnText"></div>
			<div class="cell-item-cls vertic" label="<%=rb.getString("DaiKuan")%>">${bandWidth}(MHz)</div>
			
			<div class="cell-item-cls vertic" label="CINR1">${CINR1}</div>
			<div class="cell-item-cls vertic" label="CINR2">${CINR2}</div>
			<div class="cell-item-cls vertic" label="SINR">${SINR}</div>
			<div class="cell-item-cls vertic" label="UL_MCS">${UL_MCS}</div>
			<div class="cell-item-cls vertic" label="DL_MCS">${DL_MCS}</div>
			
			<div class="cell-item-cls vertic" label="<%=rb.getString("HostName")%>">${cellName}</div>
			<div class="cell-item-cls vertic" label="<%=rb.getString("CPETxPower")%>">${txPower}</div>
			<div class="cell-item-cls vertic" label="RSRP1">
				<span v-show="'${RSRP1}'!==''" style="display: flex;align-items: center;">
					<i class="el-icon el-icon-signal small-icon signal-high" v-if="rsrp_1_value > highVal"></i>
					<i class="el-icon el-icon-signal small-icon signal-low" v-if="rsrp_1_value < lowVal"></i>
					<i class="el-icon el-icon-signal small-icon signal-normal" v-if="rsrp_1_value <= highVal && rsrp_1_value >= lowVal"></i>
					${RSRP1} dBm</span>
			</div>
			<div class="cell-item-cls vertic" label="RSRP2">
				<span v-show="'${RSRP2}'!==''" style="display: flex;align-items: center;">
					<i class="el-icon el-icon-signal small-icon signal-high" v-if="rsrp_2_value > highVal"></i>
					<i class="el-icon el-icon-signal small-icon signal-low" v-if="rsrp_2_value < lowVal"></i>
					<i class="el-icon el-icon-signal small-icon signal-normal" v-if="rsrp_2_value <= highVal && rsrp_2_value >= lowVal"></i>
						${RSRP2} dBm</span>
			</div>
		</div>
	</div>
	<div v-show="!isR005 && !is43XAP" class="infoItem enbInfo">
		<div class="infoTitle">NR <%=rb.getString("ZhuangTai")%></div>
		<div class="cellItemBoxCls">
			<div class="cell-item-cls vertic" label="Band">{{nrStatusData.NR_BAND}}</div>
			<div class="cell-item-cls vertic" label="PCI">{{nrStatusData.NR_PCI}}</div>
			<div class="cell-item-cls vertic" label="<%=rb.getString("PinDian")%>(MHz)">{{nrStatusData.NR_EARFCN}}</div>
			<div class="cell-item-cls vertic" label="PLMN">{{nrStatusData.NR_PLMN}}</div>
			<div class="cell-item-cls vertic" label="<%=rb.getString("DaiKuan")%>(MHz)">{{nrStatusData.NR_BANDWIDTH}}</div>
			<div class="cell-item-cls vertic" label="Cell_ID">{{nrStatusData.NR_CELL_ID}}</div>
			<div class="cell-item-cls vertic" label="DL_Frequency">{{nrStatusData.NR_DL_FREQUENCY}}</div>
			<div class="cell-item-cls vertic" label="UL_Frequency">{{nrStatusData.NR_UL_FREQUENCY}}</div>
			<div class="cell-item-cls vertic" label="CINR">{{nrStatusData.NR_CINR}}</div>
			<div class="cell-item-cls vertic" label="SINR">{{nrStatusData.NR_SINR}}</div>
			<div class="cell-item-cls vertic" label="RSRQ">{{nrStatusData.NR_RSRQ}}</div>
			<div class="cell-item-cls vertic" label="RSRP">{{nrStatusData.NR_RSRP}}</div>
		</div>
	</div>
	<div v-show="!isR005 && !is43XAP" class="infoItem enbInfo halfItem">
		<div class="infoTitle">LAN <%=rb.getString("ZhuangTai")%></div>
		<div class="cellItemBoxCls">
			<div class="cell-item-cls vertic" label="<%=rb.getString("MACDiZhi")%>">${macAddress}</div>
			<div class="cell-item-cls vertic" label="<%=rb.getString("LgwIPAddress")%>">${lgwIp}</div>
			<div class="cell-item-cls vertic" label="<%=rb.getString("LgwMacAddress")%>">${lgwMac}</div>
			<div class="cell-item-cls vertic" label="LAN Interface" v-html="LANLnterface"></div>
		</div>
	</div>
	<div v-show="!isR005 && !is43XAP" class="infoItem enbInfo halfItem">
		<div class="infoTitle" style="position:relative;">
			WAN <%=rb.getString("ZhuangTai")%>
			<div class="circleIcon" style="top:-5px;right:0;">
				<i class="el-icon el-icon-circle-refresh" @click="refreshWanStatus"></i>
			</div>
		</div>
		<div class="cellItemBoxCls">
			<div class="cell-item-cls vertic" label="<%=rb.getString("WanIPDiZhi")%>">{{wanIpAddress}}</div>
			<div class="cell-item-cls vertic" label="<%=rb.getString("WanZiWangYanMa")%>">{{wanSubnetMask}}</div>
			<div class="cell-item-cls vertic" label="<%=rb.getString("WanWangGuan")%>">{{wanGatewayAddress}}</div>
		</div>
	</div>
	
	<div v-show="!isR005 && !is43XAP" class="infoItem enbInfo halfItem">
		<div v-if="wanShow">
			<div class="infoTitle">WLAN <%=rb.getString("ZhuangTai")%></div>
			<div class="cellItemBoxCls">
				<div class="cell-item-cls vertic" label="${wifi1Ssid}">{{wifiStatus1}}</div>
				<div class="cell-item-cls vertic" label="${wifi2Ssid}">{{wifiStatus2}}</div>
				<div class="cell-item-cls vertic" label="${wifi3Ssid}">{{wifiStatus3}}</div>
				<div class="cell-item-cls vertic" label="${wifi4Ssid}">{{wifiStatus4}}</div>
			</div>
		</div>

		<div :class="wanShow ? '' : 'locationInfoBoxCls' ">
			<div class="infoTitle"><%=rb.getString("WeiZhi")%></div>
			<div class="cellItemBoxCls">
				<div class="cell-item-cls vertic" label="<%=rb.getString("JingDu")%>">${longitude}</div>
				<div class="cell-item-cls vertic" label="<%=rb.getString("WeiDu")%>">${latitude}</div>
				<div class="cell-item-cls vertic" label="<%=rb.getString("GaoDu")%>">${height}</div>
				<div class="cell-item-cls vertic" label="<%=rb.getString("JuLi")%>">${distance}</div>
			</div>
		</div>
	</div>
	
</div>
<script type="text/javascript">
var timeParam = getNowTimeToZoneTimeRange(timeZone, 144)
var lowVal = localStorage.getItem("rsrp1");
var highVal = localStorage.getItem("rsrp2");
var cpeCode = sessionStorage.getItem('CPE_CODE');
var globCpeMap,
	cpeDetailIMSI = '${IMSI}';
var sn = "${sn}";
var product = "${product}";
var cpeDetailVue = new Vue({
	el:'#cpeDetailPage',
	data(){
		var vm = this;
		return {
			rowData:{
				serialNumber:'${serialNumber}',
				cpeName:'${cpeName}'
			},
			rsrp_1_value:parseInt('${RSRP1}'),
			rsrp_2_value:parseInt('${RSRP2}'),
			lowVal:lowVal,
			highVal:highVal,
			wanIpAddress: '${wanIpAddress}',
			wanSubnetMask: '${wanSubnetMask}',
			wanGatewayAddress: '${wanGatewayAddress}',
			nrStatusData:{
				NR_BAND:'',
				NR_BANDWIDTH:'',
				NR_PCI:'',
				NR_EARFCN:'',
				NR_PLMN:'',
				NR_CELL_ID:'',
				NR_DL_FREQUENCY:'',
				NR_UL_FREQUENCY:'',
				NR_CINR:'',
				NR_SINR:'',
				NR_RSRQ:'',
				NR_RSRP:'',
			},
			
		}
	},
    computed: {
		modelName() {
			var value = '${modelName}',
				capablity = '${CAPABILITY}',
				lte_turbo_enalbe = '${LTE_TURBO_ENABLE}';
				
			if(capablity == '0' && isLWAEnable){
				value = "<span class='cpeLwaWu' style='font-size:18px'>"+"</span>"+ "<span style='font-size:12px;padding-top: 5px;'>"+(value)+"</span>"
			}else if(capablity =='1' && lte_turbo_enalbe == '1' && isLWAEnable){
				value = "<span class='cpeLwaKai' style='font-size:18px'>"+"</span>"+ "<span style='font-size:12px;padding-top: 5px;'>"+(value)+"</span>"
			}else if(capablity == '1' && lte_turbo_enalbe == '0' && isLWAEnable){
				value = "<span class='cpeLwaGuan' style='font-size:18px'>"+"</span>"+ "<span style='font-size:12px;padding-top: 5px;'>"+(value)+"</span>"
			}
			
			return value;
		},
		wanShow() {
			var vm = this,
				reg = /\S*EP3011\S*/,
				product = sessionStorage.getItem('oldProduct');

			return reg.test(product);
		},
		moduleFmt() {
			var value = '${module}',
				reg = new RegExp("^IDU");
			
			if (value == "LTE WiFi VoIP Gateway" || (reg.test(value)==true)) {
				return "IDU";
			} else {
				return "ODU";
			}
		},
		LANLnterface(){
			var status = '${LANLnterface}';
			
			if(status == '1' || status == '0') {
				return status == '1'? '<%=rb.getString("KaiQi")%>':'<%=rb.getString("GuanBi")%>';
			}else {
				return '--';
			}
		},
		wifiStatus1() {
			var status = '${wifi1Status}';

			if(status == '1' || status == '0') {
				return status == '1'? '<%=rb.getString("QiYong")%>':'<%=rb.getString("JinYong")%>';
			}else {
				return '--'
			}
		},
		wifiStatus2() {
			var status = '${wifi2Status}';

			if(status == '1' || status == '0') {
				return status == '1'? '<%=rb.getString("QiYong")%>':'<%=rb.getString("JinYong")%>';
			}else {
				return '--'
			}
		},
		wifiStatus3() {
			var status = '${wifi3Status}';

			if(status == '1' || status == '0') {
				return status == '1'? '<%=rb.getString("QiYong")%>':'<%=rb.getString("JinYong")%>';
			}else {
				return '--'
			}
		},
		wifiStatus4() {
			var status = '${wifi4Status}';

			if(status == '1' || status == '0') {
				return status == '1'? '<%=rb.getString("QiYong")%>':'<%=rb.getString("JinYong")%>';
			}else {
				return '--'
			}
		},
		earfcnText() {
			var val = '${earfcn}';

			if(val) {
				val = val.split(',').map(function(item){
							return earfcnFormatter(item);
						}).join(',');
			}

			return val;
		},
		isOnline() {
			return sessionStorage.getItem('CONNECTION_STATUS') == 'On';
		},
		isR005() {
			var product = sessionStorage.getItem('oldProduct');

			return product.indexOf('R005') >= 0;
		},
		is43XAP() {
			var product = sessionStorage.getItem('oldProduct');

			return product.indexOf('Nova430X') >= 0 || product.indexOf('Neutrino430X') >= 0;
		}
	},
	watch:{},
	methods:{
		// 初始化
		init(code,mac,sn,name,status,row){
			var vm = this;

			Object.keys(vm.nrStatusData).forEach(function(key){
				vm.nrStatusData[key] = row[key] ? row[key] : '';
			});
		},
		// Wan 状态信息
		refreshWanStatus() {
			var vm = this,
				cpeCode = sessionStorage.getItem('CPE_CODE'),
				sn = '${serialNumber}',
				product = sessionStorage.getItem('PRODUCT');
			
			vm.isWifiLoading = true;
			axios.post('${ctx}/cell/CPE/queryWanInfo.action',stringify({
				cpeCode: cpeCode
			})).then(function(res){
				if(res.data) {
					vm.wanIpAddress = res.data.wanIpAddress;
					vm.wanSubnetMask = res.data.wanSubnetMask;
					vm.wanGatewayAddress = res.data.wanGatewayAddress;
				}
				vm.isWifiLoading = false;
			});
		},
		
	},
	mounted(){
		var vm = this;
		eventBus.$off("cpe-data").$on("cpe-data",this.init)
	}
})
</script> 
