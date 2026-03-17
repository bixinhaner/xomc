<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	.borderPage{
		border:1px solid #d5dcec;
		box-sizing:border-box;
		border-radius:10px;
	}
	.enbNav {
		width:200px;
		border-right:1px solid #d5dcec;
	}
	#gsmSettingPage .settingMainPage {
		flex:1 ;
		display:flex;
		flex-direction:column;
		overflow: auto;
	}
	#gsmSettingPage .el-card__body {
		flex:1 auto;
		overflow:auto;
	}
	#gsmSettingPage .navItem {
		height:40px;
		line-height:40px;
		border-bottom:1px solid #d5dcec;
		padding:0 20px;
		font-size:14px;
		cursor:pointer;
	}
	#gsmSettingPage .navItem[disabled='disabled']{
		color:rgba(0,0,0,0.4);
		cursor:not-allowed;
	}
	#gsmSettingPage .navItem:hover ,#gsmSettingPage .navItem.active  {
		background-color: rgba(var(--main-color-rgba1),0.08);
		color:var(--main-color);
	}
	#gsmSettingPage.navItem .el-icon ,#gsmSettingPage .navItemTitle .el-icon{
		margin-right:10px;
		color:unset;
	}
	#gsmSettingPage .navItem .el-icon:before {
		color:unset;
	}
	#gsm_setting_main {
		padding:10px;
		background:#f1f1f2;
	}
	#gsmSettingPage .navItemGroup {
		border-bottom:1px solid #d5dcec;
	}
	#gsmSettingPage .navItemTitle {
		color:#999;
		padding: 15px 20px 10px 20px;
		font-size:14px;
	}
	#gsmSettingPage .navItemGroup  .navItem {
		border:none;
	}
	#gsmSettingPage .navItemGroup .navItem {
		padding-left:48px;
	}
	#gsmSettingPage .navItemGroup.advanceGroup .navItem {
		padding-left:20px;
	}
	
	#gsmSettingPage .left-title-list li{
		height:36px;
		line-height:36px;
		padding:0 15px;
		cursor:pointer;
		border-bottom:1px solid #EEE;
		word-break:keep-all;
	}
	#gsmSettingPage .left-title-list li.active{
		color:#4D84FF;
		background:#EDF6FF;
		font-weight:bold;
	}
	
	#gsmSettingPage .el-collapse-item__header{
		border-bottom:1px solid #fff;
		max-width: 800px;
	}
	#gsmSettingPage .el-collapse-item__arrow{
		position:absolute;
		left:20px;
		top:-1px;
	}
	#gsmSettingPage .el-collapse-item{
		position:relative;
	}
	#gsmSettingPage .el-collapse-item__header .el-icon-arrow-right{
		font-size:16px;
	}
	#gsmSettingPage .settingMainPage .el-collapse-item__header .el-icon-arrow-right:before{
		content:"\e639";
		color:#BBB;
	}
	#gsmSettingPage .settingMainPage .el-collapse-item__header .is-active.el-icon-arrow-right:before{
		content:"\e638";
		color:#BBB;
	}
	#gsmSettingPage .el-collapse-item__arrow.is-active,.gsmConfigAddDialog .el-collapse-item__arrow.is-active{
		transform:rotate(0deg);
	}
	#gsmSettingPage .el-collapse{
		border-top:1px solid #fff;
		border-bottom:1px solid #fff;
	}
	#gsmSettingPage .el-collapse-item__wrap{
		border-bottom:1px solid #EEE;
		padding-left:70px;
	}
	#gsmSettingPage .item-title-cls{
		font-weight:bold;
		position:relative;
		padding-left:15px;
		margin-bottom:20px;
	}
	#gsmSettingPage .item-title-cls:before{
		content:' ';
		width:6px;
		height:6px;
		background:#000;
		border-radius:6px;
		display:inline-block;
		position:absolute;
		top:8px;
		left:0px;
	}
	#gsmSettingPage .list-cls{
		display:flex;
	}
	#gsmSettingPage .list-cls .el-form-item{
		width:100%;
	}
	#gsmSettingPage .list-item-cls{
		flex:1;
		display:flex;
		flex-direction:column;
	}
	#gsmSettingPage .suffixItem{
		display:inline-block;
		margin-bottom:0px;
		margin-right:5px;
	}
	#gsmSettingPage .suffixItem .el-form-item__content{
		line-height:16px;
	}
	#gsmSettingPage .suffixItem .form-suffix{
		width:85px;
	}
	#gsmSettingPage .suffixItem .form-suffix .text{
		width:55px;
	}
	#gsmSettingPage .el-form--inline .el-form-item{
		margin-right:0px;
		margin-left:0px !important;
		width: 45%;
	}
	#gsmSettingPage .item-tip{
		color:#999;
		margin-left:10px;
	}
	#gsmSettingPage .el-input-group__append{
		border-radius:0px;
		width:auto;
	}
	#gsmSettingPage .mmeSelect .el-input,.mmeSelect .el-input__inner{
		width:100px !important;
	}
	#gsmSettingPage .validate-item .el-input__inner{
		width:200px !important;
	}
	.validate-item .el-input-group__append{
		border:none;
		background:none;
	}
	.validate-item .el-form-item__error{
		display:none;
	}
	.is-error .el-input-group__append,.is-error .item-tip{
		color:#FA5555;
	}
	.addSlide{
		position:absolute;
		left:0px;
		right:0px;
		top:0px;
		bottom:0px;
		background:#fff;
		z-index:100;
	}
	#gsmSettingPage .el-form-item__error{
		width:200px;
		top:3px;
		left:210px;
	}
	#gsmSettingPage .el-icon-sas-warning:before {
		color:#ff4614 !important;
	}
	#gsmSettingPage .normalText {
		color:#7a7992;
		font-weight:normal;
		margin-right: 10px;
	}
	#gsmSettingPage .navItem .el-icon {
		margin-right: 10px;
	}
	#gsmSettingPage .validate-item .el-input__inner,
	.gsmConfigAddDialog  .validate-item .el-input__inner{
		width:200px;
	}
	#gsmSettingPage .validate-item .el-input-group__append,
	.gsmConfigAddDialog .validate-item .el-input-group__append{
		border:none;
		background:none;
		padding: 0px 10px;
	}
	#gsmSettingPage .validate-item .el-form-item__error, 
	.gsmConfigAddDialog .validate-item .el-form-item__error{
		display:none;
	}
	#gsmSettingPage .is-error .el-input-group__append,
	.gsmConfigAddDialog .is-error .el-input-group__append{
		color:#FA5555;
	}
	#gsmSettingPage .is-error .el-input-group__append,
	.gsmConfigAddDialog .is-error .el-input-group__append{
		color:#FA5555;
	}
	.gsmConfigAddDialog .el-form{
		display: flex;
		flex-wrap: wrap;
		justify-content: space-between;
		width: 100%;
		position: relative;
	}
	.gsmConfigAddDialog .el-form-item{
		display: inline-block;
		width: 48%;
	}
	.gsmConfigAddDialog .el-form-item .el-form-item__label{
		font-size: 12px;
	}
	.gsmConfigAddDialog .el-form-item__error{
		padding-top: 0px;
		top:30px!important;
	}
	.gsmConfigAddDialog .selectErrcCls .el-form-item__error{
		position: absolute;
		left: 210px;
		top:5px !important;
	}
</style>
<div id="gsmSettingPage" class="borderPage" style='display:flex;flex-direction:row;flex:1;height:100%;overflow:hidden;'>
	<div class="enbNav">
		<div class="navItem" :class="currentItem === 'info' ? 'active':''"  @click="changeMain('info')">
			<span class="el-icon el-icon-overview"></span><%=rb.getString("ZongLan")%>
		</div>
        <div class="navItem" :class="currentItem === 'alarm' ? 'active':''" @click="changeMain('alarm')">
			<span class="el-icon el-icon-menu-alarm"></span><%=rb.getString("GaoJing")%>
		</div>
		<div class="navItemGroup advanceGroup" v-if="isWritable">
			<div class="navItemTitle"> 
				<span class='el-icon el-icon-operation-settings' style='color: rgba(0,0,0,0.8); cursor: text;'></span>
				<span class='commonNotes14'><%=rb.getString("SheZhi")%></span>
			</div>
			<div class="navItem" :class="currentItem === 'basic' ? 'active':''" @click="changeMain('basic')" style='padding-left: 44px;'>
				<%=rb.getString("ENBJiChuPeiZhi")%>
			</div>
           <!-- 2024-06-19 设备侧新的BSC版本没有Bts菜单了 故隐藏BTS菜单 问题单#85240-->
			<div v-if="false" class="navItem" :class="currentItem === 'bts' ? 'active':''" @click="changeMain('bts')" style='padding-left: 44px;'>
				BTS
			</div>
            <div v-if="product == 'BTS'" class="navItem" :class="currentItem === 'btsSetting' ? 'active':''" :disabled="optDisFlag" @click="changeMain('btsSetting')" style='padding-left: 44px;'>
				BTS Setting
			</div>
		</div>
		<div v-if="isWritable" class="navItem" :class="currentItem === 'upgrade' ? 'active':''" @click="changeMain('upgrade')">
			<span class="el-icon el-icon-status-kpi-normal" style='font-size: 18px;'></span><%=rb.getString("BanBenGuanLi")%>
		</div>
		<div v-if="false" class="navItem" :class="currentItem === 'backup' ? 'active':''" @click="changeMain('backup')">
			<span class="el-icon el-icon-circle-backup" style='font-size: 18px;'></span><%=rb.getString("beiFenYuHuiFu")%>
		</div>
        <div class="navItemGroup advanceGroup">
			<div class="navItemTitle"> 
				<span class='el-icon el-icon-operation-accesscontrol' style='color: rgba(0,0,0,0.8); cursor: text;'></span>
				<span class='commonNotes14'><%=rb.getString("GaoJiSheZhi")%></span>
			</div>
			<div class="navItem" :class="currentItem === 'license' ? 'active':''" @click="changeMain('license')" style='padding-left: 44px;'>
				<span><%=rb.getString("License")%></span>
				<span class="el-icon el-icon-sas-warning" style='margin-left: 10px; color: #000000;' v-if='false'></span>
			</div>			
		</div>
	</div>
	<div class="settingMainPage">
		<div class="el-card__header" style="min-height: 40px;">
			<span>{{title}}</span>
			<span class="normalText" style="margin-right:8px;">(<%=rb.getString("BSCBianMa")%> : <b>{{sn}}</b></span>
			<span class="normalText"><%=rb.getString("BSCMingCheng")%> : <b>{{name}}</b>)</span>
            <div class="el-icon-common-refresh el-icon" v-if='currentItem == "license"' @click='gsmRefreshLicenseTab' style="position:absolute;right:50px;top: 2px;font-size:20px;"></div>
			<span class="el-icon el-icon-close" @click="closeSetting"></span>
		</div>
		<div id="gsm_setting_main" class="el-card__body" style="position: relative;">
		</div>
	</div>
	
</div>

<script>
var gsmSettingVue = new Vue({
	el:'#gsmSettingPage',
	data(){
		return{
            rowData:'',
			small_cell_code:'',
			status:'',
			sn:'',
			version:"",
			product:"",
			selectedRow: '',
			title:'<%=rb.getString("ZongLan")%>',
			name:'',
			delay_avaliable:'',
			currentItem:'info',
            optDisFlag:false,
			source:'' // 记录来源：'monitor' 表示从监控页面跳转
		}
	},
	computed: {
		isWritable() {
            return writableMap.CODE_ENB_SETTINGS == true;
        }
	},
	methods:{
		// 初始化
		init(row,page,source){
			var vm = this,
				code = row.small_cell_code,
				sn = row.serial_number,
				version = row.software_version;
				status = row.connection_status;

			vm.rowData = row;
			vm.small_cell_code = code;
			vm.status = status;
			vm.sn = sn;
			vm.name = row.host_name;
			vm.version = version;
			vm.product = row.product;
			vm.delay_avaliable = row.delay_avaliable;
			vm.source = source || ''; // 记录来源
            vm.optDisFlag = status == 'Off'  ? true : false;
            if(page == 'alarm'){
				vm.changeMain(page);	
			}else{
				vm.changeMain('info');	
			}
		},
		changeMain(type){
			var vm = this;
			var code = vm.small_cell_code;
			var urlList = {
				info:'${ctx}/cell/cpeinfos/toGSMMonitorSettingOverviewPages.action?smallCellCode='+code+'&connection_status=' + vm.status + '&timeZone=' + timeZone,
				upgrade:'${ctx}/cell/cpeinfos/toGSMMonitorSettingUpgradePages.action',
				backup:'${ctx}/cell/cpeinfos/toGSMMonitorSettingBackupRestorePages.action',
				bts:'${ctx}/cell/cpeinfos/toGSMMonitorSettingBtsPages.action',
                btsSetting: '${ctx}/cell/cpeinfos/toGSMMonitorSettingBtsSettingPages.action',
				basic:'${ctx}/cell/cpeinfos/toGSMMonitorSettingBasicPages.action',
                license: '${ctx}/cell/cpeinfos/toGSMMonitorSettingLicensePage.action',
                alarm: '${ctx}/cell/cpeinfos/toGSMMonitorSettingAlarmPage.action',
			};
			
			var titleList = {
					info:'<%=rb.getString("ZongLan")%>',
					upgrade:'<%=rb.getString("ShengJi")%>',
					backup:'<%=rb.getString("beiFenYuHuiFu")%>',
					bts:'BTS',
                    btsSetting:'BTS Setting',
					basic:'<%=rb.getString("ENBJiChuPeiZhi")%>',
                    license:'<%=rb.getString("License")%>',
                    alarm:'<%=rb.getString("GaoJing")%>',
				}
			
			if(vm.optDisFlag && (type == 'btsSetting')){
				return;
			}
			
			$('#gsm_setting_main').html('');
            $('#gsm_setting_main').addClass('loading');
			loadHTML(document.querySelector('#gsm_setting_main'),{
                url:urlList[type] ,
                method:'post',
                success: function() {
                	vm.currentItem = type;
                	vm.title = titleList[type];
                	eventBus.$emit("gsm-data",vm.rowData)
					$('#gsm_setting_main').removeClass('loading'); 
                }
            });
		},
		closeSetting(){
			var vm = this;
			// 根据来源动态关闭设置页面
			if(vm.source === 'monitor'){
				// 从监控页面跳转过来的，触发监控页面的关闭事件
				eventBus.$emit('cancel-gsm-setting');
			}else{
				// 其他来源，可以添加其他处理逻辑
				eventBus.$emit('cancel-enb-topo-setting');
			}
		},
        //license Tab 右上角刷新
        gsmRefreshLicenseTab(){
            var vm = this,
                params = {
                    smallCellCode: vm.rowData.small_cell_code
                };

            // 发起同步指令- 刷新页面内容,分两种情况
            axios.post("${ctx}/cell/quicksettings/syncLicense.action",stringify(params)).then(function(res){
                var data = res.data;
                if(data["success"]){
					vm.changeMain('license');
				}else{
					vm.$message.error(data["message"])
				}
            })
        },
	},
	mounted(){
		eventBus.$off("gsm-setting-row").$on("gsm-setting-row",this.init);
		eventBus.$off("gsm-close-setting").$on("gsm-close-setting",this.closeSetting);
	}
})
</script>