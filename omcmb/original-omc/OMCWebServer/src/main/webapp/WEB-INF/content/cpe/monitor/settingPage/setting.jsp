<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	#cpeSettingPage{
		border:1px solid #d5dcec;
		box-sizing:border-box;
		border-radius:10px;
	}
	#cpeSettingPage .enbNav {
		width:200px;
		min-width: 200px;
		border-right:1px solid #d5dcec;
	}
	#cpeSettingPage .settingMainPage {
		flex:1 ;
		display:flex;
		flex-direction:column;
		overflow: auto;
	}
	#cpeSettingPage .settingMainPage .el-card__header {
		min-height:40px;
	}
	#cpeSettingPage .el-card__body {
		flex:1 auto;
		overflow:auto;
	}
	#cpeSettingPage .navItem {
		height:40px;
		line-height:40px;
		border-bottom:1px solid #d5dcec;
		padding:0 20px;
		font-size:14px;
		cursor:pointer;
	}
	#cpeSettingPage .navItem .el-icon ,#cpeSettingPage .navItemTitle .el-icon{
		margin-right:10px;
		color:unset;
	}
	#cpeSettingPage .navItem .el-icon:before {
		color:unset;
	}
	#setting_main_cpe {
		padding:10px;
		background:#f1f1f2;
	}
	#cpeSettingPage .navItemGroup {
		border-bottom:1px solid #d5dcec;
	}
	#cpeSettingPage .navItemTitle {
		color:#999;
		padding: 15px 20px 10px 20px;
		font-size:14px;
	}
	#cpeSettingPage .navItemGroup  .navItem {
		border:none;
	}
	#cpeSettingPage .settingGroup .navItem {
		padding-left:48px;
	}
	#cpeSettingPage .normalText {
		color:#7a7992;
		font-weight:normal;
	}
	#cpeSettingPage .el-collapse-item__header,.cpeConfigAddDialog .el-collapse-item__header{
		border-bottom:1px solid #fff;
	}
	#cpeSettingPage .el-collapse-item__arrow,.cpeConfigAddDialog .el-collapse-item__arrow{
		position:absolute;
		left:20px;
		top:0px;
	}
	#cpeSettingPage .el-collapse-item,.cpeConfigAddDialog .el-collapse-item{
		position:relative;
		border-bottom:1px solid #E9E9E9;
	}
	#cpeSettingPage .el-collapse-item__content,.cpeConfigAddDialog .el-collapse-item__content{
		margin: 0px 40px;
		padding-bottom: unset;
		position: relative;
	}
	#cpeSettingPage .el-collapse-item__header .el-icon-arrow-right,.cpeConfigAddDialog .el-collapse-item__header .el-icon-arrow-right{
		font-size:16px;
	}
	#cpeSettingPage .el-collapse-item__header .el-icon-arrow-right:before,.cpeConfigAddDialog .el-collapse-item__header .el-icon-arrow-right:before{
		content:"\e639";
		color:#BBB;
	}
	#cpeSettingPage .el-collapse-item__header .is-active.el-icon-arrow-right:before,.cpeConfigAddDialog .el-collapse-item__header .is-active.el-icon-arrow-right:before{
		content:"\e638";
		color:#BBB;
	}
	#cpeSettingPage .el-collapse-item__arrow.is-active,.cpeConfigAddDialog .el-collapse-item__arrow.is-active{
		transform:rotate(0deg);
	}
	#cpeSettingPage .el-collapse,.cpeConfigAddDialog .el-collapse{
		border-top:1px solid #fff;
		border-bottom:1px solid #fff;
	}
	.cpeConfigAddDialog .el-collapse{
		width: 100%;
	}
	#cpeSettingPage .el-collapse-item__wrap,.cpeConfigAddDialog .el-collapse-item__wrap{
		border-bottom:1px solid #fff;
		padding-left: 0px;
	}
	#cpeSettingPage .el-collapse-item__header,.cpeConfigAddDialog .el-collapse-item__header{
		width:100%;
		position: relative;
	}

	#cpeSettingPage  .allowMoreInputBoxCls{
		position: relative;
		flex: 1;
	}
	#cpeSettingPage .allowMoreInputBoxCls .allowMoreInputHeadCls{
		margin-bottom: 5px;
	}
	#cpeSettingPage .allowMoreInputBoxCls .allowMoreInputHeadCls .allowMoreInputTitleCls{
		font-size: 14px;
		color: rgba(0, 0, 0, 0.8);
	}
	#cpeSettingPage .allowMoreInputBoxCls .allowMoreInputHeadCls .allowMoreInputTipsCls{
		font-size: 14px;
		color: rgba(0, 0, 0, 0.32);
	}
	#cpeSettingPage .allowMoreInputBoxCls .allowMoreInputContentCls{
		border: 1px solid #DFE2EE;
		width: 80%;
		min-width: 600px;
		min-height: 78px;
		padding: 10px;
		border-radius: 4px;
		box-sizing: border-box;
		position: relative;
	}
	#cpeSettingPage .allowMoreInputBoxCls .allowMoreInputAddedCls{
		position: absolute;
		top: -23px;
		right: 0px;
	}
	#cpeSettingPage .allowMoreInputBoxCls .allowMoreInputFieldCls{
		display: flex;
	}
	#cpeSettingPage .allowMoreInputBoxCls .allowMoreInputFieldCls .el-input{
		width: 240px;
	}
	#cpeSettingPage .allowMoreInputBoxCls .allowMoreInputFieldCls .allowMoreInputAddBtnCls{
		height: 26px;
		width: 56px;
		display: flex;
		align-items: center;
		justify-content: center;
		color:var(--main-color);
		border: 1px solid var(--main-color);
		background:rgba(var(--main-color-rgba1),0.1);
		border-radius: 4px;
		box-sizing: border-box;
		margin-left: 10px;
		cursor: pointer;
	}
	#cpeSettingPage .allowMoreInputBoxCls .allowMoreInputFieldCls .allowMoreInputAddTipCls{
		margin-left: 10px;
	}
	#cpeSettingPage .allowMoreInputBoxCls .allowMoreInputFieldCls .allowMoreInputAddBtnCls .el-icon::before{
		font-size: 16px;
		color:var(--main-color);
	}
	#cpeSettingPage .allowMoreInputBoxCls .allowMoreInputFieldCls .allowMoreInputAddBtnCls span:nth-child(2){
		margin: 0px 3px;
	}
	#cpeSettingPage .allowMoreInputBoxCls .allowMoreInputParamsCls{
		display: flex;
		flex-wrap: wrap;
		width: 100%;
		margin-top: 10px;
	}
	#cpeSettingPage .allowMoreInputBoxCls .allowMoreInputParamsItemCls{
		height: 26px;
		display: flex;
		align-items: center;
		border: 1px solid #DFE2EE;
		border-radius: 4px;
		box-sizing: border-box;
		padding: 0px 10px;
		margin-right: 10px;
		margin-bottom: 5px;
		background: #F8F8FD;
	}
	#cpeSettingPage .allowMoreInputBoxCls .allowMoreInputParamsItemCls span:nth-child(1){
		display: inline-block;
		max-width: 520px;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	#cpeSettingPage .allowMoreInputBoxCls .allowMoreInputParamsItemCls span:nth-child(2){
		margin-left: 10px;
	}
	#cpeSettingPage .allowMoreInputBoxCls .allowMoreInputParamsItemCls .el-icon-close{
		font-size: unset;
		position: unset;
		top: unset;
		right: unset;
	}
	#cpeSettingPage .allowMoreInputBoxCls .allowMoreInputParamsItemCls .el-icon-close::before{
		font-size: 12px;
		color: #7A7992;
	}
	#cpeSettingPage .allowMoreInputBoxCls .allowMoreInputFootCls{
		height: 18px;
	}
	#cpeSettingPage .allowMoreInputBoxCls .allowMoreInputFootCls .inputErrorBoxCls{
		color:red;
		font-size:10px;
	}
</style>
<div id="cpeSettingPage" class="borderPage" style='display:flex;flex-direction:row;flex:1;height:100%;overflow:hidden;'>
	<div class="enbNav">
		<div class="navItem" @click="changeMain('info')">
			<span class="el-icon el-icon-overview"></span><%=rb.getString("ZongLan")%>
		</div>
		<div class="navItem" @click="changeMain('chart')" v-show="!isR005 && !is43XAP">
			<span class="el-icon el-icon-operation-statistics"></span><%=rb.getString("TongJi")%>
		</div>
		
		<div class="navItemGroup settingGroup CODE_CPE_SETTINGS hidden">
			<div class="navItemTitle"><span class="el-icon el-icon-operation-settings"></span><%=rb.getString("SheZhi")%></div>
			<div class="navItem" v-for="(item,index) in tabs" v-show="item.show" :class="{active: settingTab == item.code, disabled: (!isOnline && index>0)}" @click="changeToSetting(item)">
				<span :class="item.class"></span>{{item.text}}
			</div>
			
		</div>
		
		<div v-if="hasUpgradeRole" class="navItem" @click="changeMain('upgrade')">
			<span class="el-icon el-icon-menu-upgrade"></span><%=rb.getString("ShengJi")%>
		</div>
		<div class="navItem" @click="changeMain('log')" v-show="!isR005 && !is43XAP">
			<span class="el-icon el-icon-operation-details"></span><%=rb.getString("RiZhi")%>
		</div>
		<div class="navItemGroup CODE_CPE_MONITOR hidden" v-show="!isR005 && !is43XAP">
			<div class="navItemTitle"> <%=rb.getString("GaoJiSheZhi")%></div>
			<div class="navItem" @click="changeMain('speed')">
				<span class="el-icon el-icon-operation-diagnostic"></span> <%=rb.getString("CeSu")%>
			</div>
			<div v-if="is4G" class="navItem" @click="changeMain('scan')">
				<span class="el-icon el-icon-operation-scan"></span> <%=rb.getString("ZhanDianSaoMiao")%>
			</div>
		</div>
		
	</div>
	<div class="settingMainPage">
		<div class="el-card__header">
			<span>{{title}}</span>
			<span class="normalText" style="margin-right:8px;">(<%=rb.getString("CPEXuLieHao")%> : <b>{{sn}}</b></span>
			<span class="normalText"><%=rb.getString("CPEName")%> : <b>{{name}}</b>)</span> 
			<span class="el-icon el-icon-close" @click="closeSetting"></span>
		</div>
		<div id="setting_main_cpe" class="el-card__body" style="position: relative;">
		</div>
		<div v-if="false" class="el-card__footer">
		</div>
	</div>
	
</div>

<script>
var cpeSettingVue = new Vue({
	el:'#cpeSettingPage',
	data(){
		return{
			code:'',
			sn:'',
			name:'',
			status:'',
			mac:'',
			rowData:'',
			sourceFrom:'', // 记录从哪个页面跳转过来的：'cpe_monitor' 或 'cpe_topo'
			
			title:'<%=rb.getString("ZongLan")%>',
			settingTab: '',
			isOnline: false,
			is4G: false,
		}
	},
	computed: {
		tabs() {
			var product = sessionStorage.getItem('oldProduct'),
				isR005 = product.indexOf('R005') >= 0,
				is43XAP = product.indexOf('Nova430X') >= 0 || product.indexOf('Neutrino430X') >= 0;

			return [
				{code: 'basic', text: '<%=rb.getString("JiBenSheZhi")%>',show: true},
				{code: 'wifi', text: 'WIFI Config',show: isR005 && !is43XAP},
				{code: 'network', text: '<%=rb.getString("WangLuoSheZhi")%>',show: true},
				{code: 'lte', text: 'LTE/NR',show: !this.wanShow && !isR005 && !is43XAP},
				{code: 'system', text: '<%=rb.getString("XiTong")%>',show: true && !isR005 && !is43XAP},
				{code: 'apnSet', text: 'APN/L2 <%=rb.getString("SheZhi")%>',show: this.apnSetShow && !isR005 && !is43XAP}
			]
		},
		wanShow() {
			var vm = this,
				reg = /\S*EP3011\S*/,
				product = sessionStorage.getItem('oldProduct');

			return reg.test(product);
		},
		apnSetShow(){
			var vm = this,
				l2flag = false,
				reg = new RegExp("^(IDU\/CN)"),
				regIduEG = new RegExp("^(IDU\/EG)"),
				regOduEG = new RegExp("^(ODU\/EG)"),
				regu4G = new RegExp("^((ODU\/u4G)|(IDU\/u4G))"),
				product = sessionStorage.getItem('oldProduct');

			if (product == "LTE WiFi VoIP Gateway" || (reg.test(product)==true)) {
				l2flag = false;
			} else if(regIduEG.test(product)){
				l2flag = true;
			} else if(regOduEG.test(product)){
				l2flag = true;
			} else if(regu4G.test(product)){
				l2flag = true;
			} else {
				l2flag = false;
			}

			if(l2flag && writableMap["CODE_CPE_APN"] == true){
				return true
			} else {
				return false
			}
		},
		hasUpgradeRole() {
			return writableMap.CODE_CPE_UPGRADE_FILE == true || writableMap.CODE_CPE_UPGRADE_IMAGE == true;
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
	methods:{
		// 初始化
		init(code,mac,sn,status,row,sourceFrom){
			var vm = this;
			vm.code = code;
			vm.status = status;
			vm.mac = mac;
			vm.sn = sn;
			vm.name = sessionStorage.getItem('cpeName');
			vm.rowData = row;
			vm.sourceFrom = sourceFrom || 'cpe_monitor'; // 记录来源页面，默认为 cpe_monitor

            vm.isOnline = row.CONNECTION_STATUS == 'On' ? true : false;
			vm.is4G = row.cpe_model == '4G' ? true : false;

			vm.changeMain('info');
		},
		changeMain(type){
			var vm = this;
			
			var urlList = {
				info:"${ctx}/cell/CPE/toCpeDetailParamInfoPage.action?cpeCode=" + vm.code+'&timeZone='+timeZone,
				chart:'${ctx}/cpe/setting/openStatisticPage.action',
				setting:'${ctx}/cell/CPE/toCpeSettingPage.action',
				upgrade:'${ctx}/cpe/setting/openUpgradePage.action',
				log:'${ctx}/cpe/setting/openLogPage.action',
				speed:'${ctx}/cpe/diagnostics/toDiagnosticsPage.action',
				scan: '${ctx}/cpe/setting/openCellScan.action',
			};
			
			var titleList = {
				info:'<%=rb.getString("ZongLan")%>',
				chart:'<%=rb.getString("TongJi")%>',
				setting:'',
				upgrade:'<%=rb.getString("ShengJi")%>',
				log:'<%=rb.getString("RiZhi")%>',
				speed:'<%=rb.getString("CeSu")%>',
				scan: 'Cell Scan'
			}
			
			$('#setting_main_cpe').html('');
            $('#setting_main_cpe').addClass('loading');
			
			loadHTML(document.querySelector('#setting_main_cpe'),{
                url:urlList[type] ,
                success: function() {
                	vm.title = titleList[type];
                	eventBus.$emit("cpe-data",vm.code,vm.mac,vm.sn,vm.name,vm.status,vm.rowData);
                	$('#setting_main_cpe').removeClass('loading');
                }
            });
		},
		changeToSetting(item){
			var url = '${ctx}/cell/CPE/toCpeSettingPage.action';
			var vm = this;
			
			var titleList = {
				basic:'<%=rb.getString("JiBenSheZhi")%>',
				network:'<%=rb.getString("WangLuoSheZhi")%>',
				lte:'LTE/NR',
				system:'<%=rb.getString("XiTong")%>',
				apnSet:'APN/L2 <%=rb.getString("SheZhi")%>',
			}
			if(item.code == 'basic'){
				url = '${ctx}/cell/CPE/setting/goToBasicSetting.action';
			}else if(item.code == 'lte'){
				url = '${ctx}/cell/CPE/toLteOrNrPage.action';
			}else if(item.code == 'wifi') {
				url = '${ctx}/cell/CPE/toWifiPage.action';
			}
			if(!vm.isOnline && item.code != 'basic') return;
			/*
			loadHTML(document.querySelector('#setting_main_cpe'),{
                url:url ,
                success: function() {
                	eventBus.$emit("cpe-data",item,vm.code)
                	vm.settingTab = item.code;
                	vm.title = titleList[item.code];
                }
            });
			*/
			$('#setting_main_cpe').html('');
			$('#setting_main_cpe').addClass('loading');
			$('#setting_main_cpe').load(url,function(){
				eventBus.$emit("cpe-data",item,vm.code,vm.rowData)
				vm.settingTab = item.code;
				vm.title = titleList[item.code];
				setTimeout(function(){
					$('#setting_main_cpe').removeClass('loading');
				},200);
			})
		},
		closeSetting(){
			var vm = this;
			// 根据来源页面，触发不同的关闭事件
			if(vm.sourceFrom === 'topo') {
				// 从 Topo 页面跳转过来，触发 Topo 页面的关闭方法
				eventBus.$emit('cancel-enb-topo-setting');
			} else {
				// 从Monitor 页面跳转过来（默认），触发 Monitor 页面的关闭方法
				eventBus.$emit('cancel-cpe-setting');
			}
		}
	},
	mounted(){
		var vm = this;
		eventBus.$off("row-data").$on("row-data",this.init);
		// 监听外部关闭事件（兼容旧的关闭方式）
		eventBus.$off("close-cpe-setting").$on("close-cpe-setting", vm.closeSetting);
	}
})
</script>
<script>
    function earfcnFormatter(value, rowData, rowIndex){
        if(value == null){
            return "";
        }
        var EARFCN = value;//正常显示的频点值  还需要将此值转换成频率
        var frequency = 0;
        if (EARFCN >= 36000 && EARFCN <= 36199) { //tdd-band 33
            frequency = 1900 + 0.1 * (EARFCN - 36000);
        } else if (EARFCN >= 36200 && EARFCN <= 36349) { //tdd-band 34
            frequency = 2010 + 0.1 * (EARFCN - 36200);
        } else if (EARFCN >= 36350 && EARFCN <= 36949) { //tdd-band 35
            frequency = 1850 + 0.1 * (EARFCN - 36350);
        } else if (EARFCN >= 36950 && EARFCN <= 37549) { //tdd-band 36
            frequency = 1930 + 0.1 * (EARFCN - 36950);
        } else if (EARFCN >= 37550 && EARFCN <= 37749) { //tdd-band 37
            frequency = 1910 + 0.1 * (EARFCN - 37550);
        } else if (EARFCN >= 37750 && EARFCN <= 38249) { //tdd-band 38
            frequency = 2570 + 0.1 * (EARFCN - 37750);
        } else if (EARFCN >= 38250 && EARFCN <= 38649) { //tdd-band 39
            frequency = 1880 + 0.1 * (EARFCN - 38250);
        } else if (EARFCN >= 38650 && EARFCN <= 39649) { //tdd-band 40
            frequency = 2300 + 0.1 * (EARFCN - 38650);
        } else if (EARFCN >= 39650 && EARFCN <= 41589) { //tdd-band 41
            frequency = 2496 + 0.1 * (EARFCN - 39650);
        } else if (EARFCN >= 41590 && EARFCN <= 43589) { //tdd-band 42
            frequency = 3400 + 0.1 * (EARFCN - 41590);
        } else if (EARFCN >= 43590 && EARFCN <= 45589) { //tdd-band 43
            frequency = 3600 + 0.1 * (EARFCN - 43590);
        } else if (EARFCN >= 18000 && EARFCN <= 18599) { //fdd-band 1
            frequency = 1920 + 0.1 * (EARFCN - 18000);
        } else if (EARFCN >= 0 && EARFCN <= 599) {
            frequency = 2110 + 0.1 * (EARFCN - 0);
        } else if (EARFCN >= 18600 && EARFCN <= 19199) { //fdd-band 2
            frequency = 1850 + 0.1 * (EARFCN - 18600);
        } else if (EARFCN >= 600 && EARFCN <= 1199) {
            frequency = 1930 + 0.1 * (EARFCN - 600);
        } else if (EARFCN >= 19200 && EARFCN <= 19949) { //fdd-band 3
            frequency = 1710 + 0.1 * (EARFCN - 19200);
        } else if (EARFCN >= 1200 && EARFCN <= 1949) {
            frequency = 1805 + 0.1 * (EARFCN - 1200);
        } else if (EARFCN >= 19950 && EARFCN <= 20399) { //fdd-band 4
            frequency = 1710 + 0.1 * (EARFCN - 19950);
        } else if (EARFCN >= 1950 && EARFCN <= 2399) {
            frequency = 2110 + 0.1 * (EARFCN - 1950);
        } else if (EARFCN >= 20400 && EARFCN <= 20649) { //fdd-band 5
            frequency = 824 + 0.1 * (EARFCN - 20400);
        } else if (EARFCN >= 2400 && EARFCN <= 2649) {
            frequency = 869 + 0.1 * (EARFCN - 2400);
        } else if (EARFCN >= 20650 && EARFCN <= 20749) { //fdd-band 6
            frequency = 830 + 0.1 * (EARFCN - 20650);
        } else if (EARFCN >= 2650 && EARFCN <= 2749) {
            frequency = 875 + 0.1 * (EARFCN - 2650);
        } else if (EARFCN >= 20750 && EARFCN <= 21449) { //fdd-band 7
            frequency = 2500 + 0.1 * (EARFCN - 20750);
        } else if (EARFCN >= 2750 && EARFCN <= 3449) { 
            frequency = 2620 + 0.1 * (EARFCN - 2750);
        } else if (EARFCN >= 3450 && EARFCN <= 3799) { //fdd-band 8
            frequency = 925 + 0.1 * (EARFCN - 3450);
        } else if (EARFCN >= 5010 && EARFCN <= 5179) { //fdd-band 12
            frequency = 729 + 0.1 * (EARFCN - 5010);
        } else if (EARFCN >= 5180 && EARFCN <= 5279) { //fdd-band 13
            frequency = 746 + 0.1 * (EARFCN - 5180);
        } else if (EARFCN >= 5730 && EARFCN <= 5849) { //fdd-band 17
            frequency = 734 + 0.1 * (EARFCN - 5730);
        } else if (EARFCN >= 6150 && EARFCN <= 6449) { //fdd-band 20     791-821 6150-6449  
            frequency = 791 + 0.1 * (EARFCN - 6150);
        }else if (EARFCN >= 9210 && EARFCN <= 9659) { //fdd-band 28       758-803 9210-9659
            frequency = 758 + 0.1 * (EARFCN - 9210);
        }else if(EARFCN >= 55240 && EARFCN <= 56740){
            frequency = 3550 + 0.1*(EARFCN - 55240);
        }else if(EARFCN >= 46790 && EARFCN <= 54539){
            frequency = 5150 + 0.1*(EARFCN - 46790);
        }else if(EARFCN >= 63000 && EARFCN <= 63999){
            frequency = 5150 + 0.1*(EARFCN - 63000);
        }else if(EARFCN >= 64000 && EARFCN <= 64999){
            frequency = 5725 + 0.1*(EARFCN - 64000);
        }else if(EARFCN >= 46790 && EARFCN <= 54539){
            frequency = 5150 + 0.1*(EARFCN - 46790);
        }else if(EARFCN >= 63000 && EARFCN <= 63999){
            frequency = 5150 + 0.1*(EARFCN - 63000);
        }else if(EARFCN >= 64000 && EARFCN <= 64999){
            frequency = 5725 + 0.1*(EARFCN - 64000);
        } else {
        //throw new Exception("Please Input the right EARFCN!");
        frequency = "--";
        }
        var showStr = EARFCN.toString()+"("+frequency.toString()+"MHz"+")";
        
        return showStr;
    }
</script>