<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	.borderPage{
		border:1px solid #d5dcec;
		box-sizing:border-box;
		border-radius:10px;
	}
	.egwNav {
		width:200px;
		border-right:1px solid #d5dcec;
	}
	.settingMainPage {
		width: calc(100% - 200px);
		flex:1 ;
		display:flex;
		flex-direction:column;
	}
	.settingMainPage .el-card__header {
		min-height:40px;
	}
	.settingMainPage .el-icon-close{
		right: 10px !important;
	}
	.settingMainPage .el-icon-close::before{
		font-size: 16px;
	}
	.el-card__body {
		flex:1 auto;
		overflow:auto;
	}
	.navItem {
		height:40px;
		line-height:40px;
		border-bottom:1px solid #d5dcec;
		padding:0 10px;
		font-size:14px;
		cursor:pointer;
	}
	.navItem .el-icon ,.navItemTitle .el-icon{
		margin-right:10px;
		color:unset;
	}
	.navItem .el-icon:before,.navItemTitle .el-icon:before{
		color:unset;
		font-size: 14px;
	}
	#setting_main_egw {
		padding:10px;
		background:#f1f1f2;
	}
	.navItemGroup {
		border-bottom:1px solid #d5dcec;
	}
	.navItemTitle {
		color:#999;
		padding: 15px 15px 10px 10px;
		font-size:14px;
	}
	.navItemGroup .navItem {
		border:none;
	}
	.settingGroup .navItem {
		padding-left:40px;
		white-space: nowrap;
		font-weight: normal;
	}
	.normalText {
		color:#7a7992;
		font-weight:normal;
	}
    .navItem:hover ,.navItem.active  {
		background-color: rgba(var(--main-color-rgba1),0.08);
		color:var(--main-color);
	}
	.newErrorCls .el-input-group__append{
		border-radius:0px;
		border-right:none;
	}
	.newErrorCls .validate-item .el-input-group__append{
		border:none;
		background:none;
	}
	.newErrorCls .validate-item .el-form-item__error{
		display:none;
	}
	.newErrorCls .is-error .el-input-group__append{
		color:#FA5555;
	}
	.newErrorCls .el-form-item__error{
		padding-top: 0px;
	}
	.newErrorCls .validate-item .el-input__inner{
		width:230px;
	}
	.gnbConfigAddDialog .gnbConfigAddMainBoxCls{
		height: 400px;
		width: 100%;
		position: relative;
		overflow-y: auto!important;
		overflow-x:hidden;
	}
	.gnbConfigAddDialog .el-dialog__header .el-icon:before{
		font-size: 16px;
	}
	.gnbConfigAddDialog .el-form{
		display: flex;
		flex-wrap: wrap;
		justify-content: space-between;
		width: 100%;
		position: relative;
	}
	.gnbConfigAddDialog .el-form-item{
		display: inline-block;
		width: 48%;
	}
	.gnbConfigAddDialog .el-form-item .el-form-item__label{
		font-size: 12px;
	}
	.gnbConfigAddDialog .el-form-item__error{
		padding-top: 0px;
		top:30px!important;
	}
	.gnbConfigAddDialog .selectErrcCls .el-form-item__error{
		position: absolute;
		left: 210px;
		top:5px !important;
	}
	#egwSettingPage .validate-item .el-input__inner,
	.gnbConfigAddDialog  .validate-item .el-input__inner{
		width:200px;
	}
	#egwSettingPage .validate-item .el-input-group__append,
	.gnbConfigAddDialog .validate-item .el-input-group__append{
		border:none;
		background:none;
		padding: 0px 10px;
	}
	#egwSettingPage .validate-item .el-form-item__error, 
	.gnbConfigAddDialog .validate-item .el-form-item__error{
		display:none;
	}
	#egwSettingPage .is-error .el-input-group__append,
	.gnbConfigAddDialog .is-error .el-input-group__append{
		color:#FA5555;
	}
	#egwSettingPage  .allowMoreInputBoxCls{
		position: relative;
		padding-left: 25px;
	}
	#egwSettingPage .allowMoreInputBoxCls .allowMoreInputHeadCls{
		margin-bottom: 5px;
	}
	#egwSettingPage .allowMoreInputBoxCls .allowMoreInputHeadCls .allowMoreInputTitleCls{
		font-size: 14px;
		color: rgba(0, 0, 0, 0.8);
	}
	#egwSettingPage .allowMoreInputBoxCls .allowMoreInputHeadCls .allowMoreInputTipsCls{
		font-size: 14px;
		color: rgba(0, 0, 0, 0.32);
	}
	#egwSettingPage .allowMoreInputBoxCls .allowMoreInputContentCls{
		border: 1px solid #DFE2EE;
		width: 80%;
		min-height: 78px;
		padding: 10px;
		border-radius: 4px;
		box-sizing: border-box;
	}
	#egwSettingPage .allowMoreInputBoxCls .allowMoreInputFieldCls{
		display: flex;
	}
	#egwSettingPage .allowMoreInputBoxCls .allowMoreInputFieldCls .el-input{
		width: 240px;
	}
	#egwSettingPage .allowMoreInputBoxCls .allowMoreInputFieldCls .allowMoreInputAddBtnCls{
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
	}
	#egwSettingPage .allowMoreInputBoxCls .allowMoreInputFieldCls .allowMoreInputAddBtnCls .el-icon::before{
		font-size: 16px;
		color:var(--main-color);
	}
	#egwSettingPage .allowMoreInputBoxCls .allowMoreInputFieldCls .allowMoreInputAddBtnCls span:nth-child(2){
		margin-left: 3px;
	}
	#egwSettingPage .allowMoreInputBoxCls .allowMoreInputParamsCls{
		display: flex;
		flex-wrap: wrap;
		width: 100%;
		margin-top: 10px;
	}
	#egwSettingPage .allowMoreInputBoxCls .allowMoreInputParamsItemCls{
		height: 26px;
		display: inline-block;
		line-height: 26px;
		border: 1px solid #DFE2EE;
		border-radius: 4px;
		box-sizing: border-box;
		padding: 0px 10px;
		margin-right: 10px;
		margin-bottom: 5px;
		background: #F8F8FD;
	}
	#egwSettingPage .allowMoreInputBoxCls .allowMoreInputParamsItemCls .el-icon-close{
		font-size: unset;
		position: unset;
		top: unset;
		right: unset;
	}
	#egwSettingPage .allowMoreInputBoxCls .allowMoreInputParamsItemCls .el-icon-close::before{
		font-size: 12px;
		color: #7A7992;
	}
	#egwSettingPage .allowMoreInputBoxCls .allowMoreInputFootCls{
		height: 18px;
	}
	#egwSettingPage .allowMoreInputBoxCls .allowMoreInputFootCls .inputErrorBoxCls{
		color:red;
		font-size:10px;
	}
</style>
<div id="egwSettingPage" class="borderPage" style='display:flex;flex-direction:row;flex:1;height:100%;overflow:hidden;'>
	<div class="egwNav">
		<div class="navItem" :class="settingTab === 'info' ? 'active':''" @click="changeMain('info')">
			<span class="el-icon el-icon-overview"></span><%=rb.getString("ZongLan")%>
		</div>
		<div class="navItem" :class="settingTab === 'chart' ? 'active':''" @click="changeMain('chart')">
			<span class="el-icon el-icon-operation-statistics"></span><%=rb.getString("TongJi")%>
		</div>
		<div class="navItem" :class="settingTab === 'alarm' ? 'active':''" @click="changeMain('alarm')">
			<span class="el-icon el-icon-menu-alarm"></span><%=rb.getString("GaoJingGuanLi")%>
		</div>
		<div class="navItemGroup settingGroup" v-show="optBtnShow">
			<div class="navItemTitle"><span class="el-icon el-icon-operation-settings"></span><%=rb.getString("SheZhi")%></div>
			<div class="navItem" v-for="(item,index) in tabs" v-show="item.show" :class="{active: settingTab == item.code, disabled: (!isOnline && index>0)}" @click="changeMain(item.code)">
				<span :class="item.class"></span>{{item.text}}
			</div>
		</div>
		<div class="navItem"  :class="settingTab === 'infoQuery' ? 'active':''" @click="changeMain('infoQuery')">
			<span class="el-icon el-icon-common-search"></span>Information Query
		</div>
		<div v-show="optBtnShow" class="navItem"  :class="settingTab === 'upgrade' ? 'active':''" @click="changeMain('upgrade')">
			<span class="el-icon el-icon-menu-upgrade"></span><%=rb.getString("ShengJi")%>
		</div>
        <div v-show="optBtnShow" class="navItem"  :class="settingTab === 'backup' ? 'active':''" @click="changeMain('backup')">
			<span class="el-icon el-icon-circle-backup"></span><%=rb.getString("beiFenYuHuiFu")%>
		</div>
		<div class="navItem"  :class="settingTab === 'log' ? 'active':''" @click="changeMain('log')">
			<span class="el-icon el-icon-operation-details"></span><%=rb.getString("RiZhi")%>
		</div>
		<div class="navItem"  :class="settingTab === 'license' ? 'active':''" @click="changeMain('license')">
			<span class="el-icon el-icon-operation-details"></span><%=rb.getString("License")%>
		</div>
	</div>
	<div class="settingMainPage">
		<div class="el-card__header">
			<span>{{title}}</span>
			<span class="normalText" style="margin-right:8px;">(<%=rb.getString("eGWBianMa")%> : <b>{{sn}}</b></span>
			<span class="normalText"><%=rb.getString("EGWMingCheng")%> : <b>{{name}}</b>)</span> 
			<span class="el-icon el-icon-close" @click="closeSetting"></span>
		</div>
		<div id="setting_main_egw" class="el-card__body" style="position: relative;">
		</div>
		<div v-if="false" class="el-card__footer"></div>
	</div>
	
</div>
<script>
if(window.egwSettingVue) {
	try {
		window.egwSettingVue.$destroy();
	}catch(e){}
}
window.egwSettingVue = new Vue({
	el:'#egwSettingPage',
	data(){
		return{
			code:'',
			sn:'',
			name:'',
			status:'',
			rowData:'',
			title:'<%=rb.getString("ZongLan")%>',
			settingTab: 'info',
			isOnline: true,
		}
	},
	computed: {
		tabs() {
			var generation = this.rowData.generation ? this.rowData.generation : '',
				isShow4G = false,
				isShow5G = false,
				isShowSeGW = false;
			if(generation == '' || generation == 'SigGW4G' || generation == 'SigGW4/5G' || generation == 'SigGW4G+SeGW' || generation == 'SigGW4/5G+SeGW'){
				isShow4G = true;
			}
			if(generation == 'SigGW5G' || generation == 'SigGW4/5G' || generation == 'SigGW5G+SeGW' || generation == 'SigGW4/5G+SeGW'){
				isShow5G = true;
			}
			if(generation == '' || generation == 'SigGW4G+SeGW' || generation == 'SigGW5G+SeGW' || generation == 'SigGW4/5G+SeGW' ){
				isShowSeGW = true;
			}
			return [
				{code: 'systemConfig', text: 'System Config',show: true},
				{code: 'securityGateway', text: 'Security Gateway',show: isShowSeGW},
				{code: 'signalingGateway_4G', text: '4G Signaling Gateway',show: isShow4G},
				{code: 'signalingGateway_5G', text: '5G Signaling Gateway',show: isShow5G},
			]
		},
		optBtnShow() {
			return writableMap['CODE_EGW'] == true;
		},
	},
	methods:{
		// 初始化
		init(row,tabType){
			var vm = this;
			vm.rowData = row;
			vm.code = row.egwCode;
			vm.status = row.connectionStatus;
			vm.sn = row.egwSn;
			vm.name = row.egwName;
			if(tabType == 'alarm'){
				vm.changeMain('alarm');
			}else{
				vm.changeMain('info');
			}
			
		},
		changeMain(type){
			var vm = this,
                urlList = {
                    info:'${ctx}/egw/setting/openOverviewPage.action',
                    chart:'${ctx}/egw/setting/openStatisticPage.action',
					alarm:'${ctx}/egw/setting/openAlarmPage.action',
                    systemConfig:'${ctx}/egw/setting/openSystemConfigPage.action',
                    securityGateway:'${ctx}/egw/setting/openSecurityGatewayPage.action',
                    signalingGateway_4G:'${ctx}/egw/setting/open4GSignalingGatewayPage.action',
                    signalingGateway_5G:'${ctx}/egw/setting/open5GSignalingGatewayPage.action',
                    infoQuery:'${ctx}/egw/setting/openInformationQueryPage.action',
                    upgrade:'${ctx}/egw/setting/openUpgradePage.action',
                    backup:'${ctx}/egw/setting/openBackupAndRestorePage.action',
                    log:'${ctx}/egw/setting/openLogPage.action',
					license:'${ctx}/egw/license/manage/toLicense.action',
                },
                titleList = {
                    info:'<%=rb.getString("ZongLan")%>',
                    chart:'<%=rb.getString("TongJi")%>',
                    systemConfig:'System Config',
                    securityGateway:'Security Gateway',
                    signalingGateway_4G:'4G Signaling Gateway',
                    signalingGateway_5G:'5G Signaling Gateway',
                    infoQuery:'Information Query',
                    upgrade:'<%=rb.getString("ShengJi")%>',
                    backup:'<%=rb.getString("beiFenYuHuiFu")%>',
                    log:'<%=rb.getString("RiZhi")%>',
					license:'<%=rb.getString("License")%>',
                };
			$('#setting_main_egw').html('');
            $('#setting_main_egw').addClass('loading');
			loadHTML(document.querySelector('#setting_main_egw'),{
                url:urlList[type] ,
                success: function() {
                    vm.settingTab = type;
                	vm.title = titleList[type];
                	eventBus.$emit("egw-data",vm.rowData,vm.code,vm.sn,vm.name,vm.status);
					$('#setting_main_egw').removeClass('loading');
                }
            });
		},
		closeSetting(){
			eventBus.$emit('cancel-egw-setting')
		}
	},
	mounted(){
		var vm = this;
		if(window.updateEgwMonitorTimer) clearInterval(window.updateEgwMonitorTimer);
		window.updateEgwMonitorTimer = setInterval(function(){
			var egwSettingPageCtn = $("#egwSettingPage");			
			if(!egwSettingPageCtn.length) {
				clearInterval(window.updateEgwMonitorTimer);
				return;
			}
			egwMonitor.$refs.egwMonitorTable.refresh();
		},6000);
		eventBus.$off("egw-setting-init").$on("egw-setting-init",this.init)
	}
})
</script>
