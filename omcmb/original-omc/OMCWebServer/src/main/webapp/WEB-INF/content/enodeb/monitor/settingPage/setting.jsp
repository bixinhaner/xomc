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
	#settingPage .settingMainPage {
		flex:1 ;
		display:flex;
		flex-direction:column;
		overflow: auto;
	}
	#settingPage .el-card__body {
		flex:1 auto;
		overflow:auto;
	}
	#settingPage .navItem {
		height:40px;
		line-height:40px;
		border-bottom:1px solid #d5dcec;
		padding:0 20px;
		font-size:14px;
		cursor:pointer;
	}
	#settingPage .navItem[disabled='disabled']{
		color:rgba(0,0,0,0.4);
		cursor:not-allowed;
	}
	#settingPage .navItem:hover ,#settingPage .navItem.active  {
		background-color: rgba(var(--main-color-rgba1),0.08);
		color:var(--main-color);
	}
	#settingPage.navItem .el-icon ,#settingPage .navItemTitle .el-icon{
		margin-right:10px;
		color:unset;
	}
	#settingPage .navItem .el-icon:before {
		color:unset;
	}
	#setting_main {
		padding:10px;
		background:#f1f1f2;
	}
	#settingPage .navItemGroup {
		border-bottom:1px solid #d5dcec;
	}
	#settingPage .navItemTitle {
		color:#999;
		padding: 15px 20px 10px 20px;
		font-size:14px;
	}
	#settingPage .navItemGroup  .navItem {
		border:none;
	}
	#settingPage .navItemGroup .navItem {
		padding-left:48px;
	}
	#settingPage .navItemGroup.advanceGroup .navItem {
		padding-left:20px;
	}
	
	#settingPage .left-title-list li{
		height:36px;
		line-height:36px;
		padding:0 15px;
		cursor:pointer;
		border-bottom:1px solid #EEE;
		word-break:keep-all;
	}
	#settingPage .left-title-list li.active{
		color:#4D84FF;
		background:#EDF6FF;
		font-weight:bold;
	}
	
	#settingPage .el-collapse-item__header{
		border-bottom:1px solid #fff;
		max-width: 800px;
	}
	#settingPage .el-collapse-item__arrow{
		position:absolute;
		left:20px;
		top:-1px;
	}
	#settingPage .el-collapse-item{
		position:relative;
	}
	#settingPage .el-icon-arrow-right{
		font-size:16px;
	}
	#settingPage .settingMainPage .el-icon-arrow-right:before{
		content:"\e639";
		color:#BBB;
	}
	#settingPage .settingMainPage .is-active.el-icon-arrow-right:before{
		content:"\e638";
		color:#BBB;
	}
	
	#settingPage .el-collapse{
		border-top:1px solid #fff;
		border-bottom:1px solid #fff;
	}
	#settingPage .el-collapse-item__wrap{
		border-bottom:1px solid #EEE;
		padding-left:70px;
	}
	#settingPage .item-title-cls{
		font-weight:bold;
		position:relative;
		padding-left:15px;
		margin-bottom:20px;
	}
	#settingPage .item-title-cls:before{
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
	#settingPage .settingMainPage .el-form-item{
		margin-bottom:20px;
		margin-left:16px;
		display:inline-block;
		width:45%;
	}
	#settingPage .settingMainPage .el-form-item{
		margin-bottom:20px;
		margin-left:16px;
		display:inline-block;
		width:95%;
	}
	#settingPage .settingMainPage .el-form-item__label{
		line-height:28px;
		font-size:12px;
		/*margin-left:15px;*/
	}
	#settingPage .list-cls{
		display:flex;
	}
	#settingPage .list-cls .el-form-item{
		width:100%;
	}
	#settingPage .list-item-cls{
		flex:1;
		display:flex;
		flex-direction:column;
	}
	#settingPage .suffixItem{
		display:inline-block;
		margin-bottom:0px;
		margin-right:5px;
	}
	#settingPage .suffixItem .el-form-item__content{
		line-height:16px;
	}
	#settingPage .suffixItem .form-suffix{
		width:85px;
	}
	#settingPage .suffixItem .form-suffix .text{
		width:55px;
	}
	#settingPage .el-form--inline .el-form-item{
		margin-right:0px;
		margin-left:0px !important;
		width: 45%;
	}
	#settingPage .item-tip{
		color:#999;
		margin-left:10px;
	}
	#settingPage .el-input-group__append{
		border-radius:0px;
		width:auto;
	}
	#settingPage .mmeSelect .el-input,.mmeSelect .el-input__inner{
		width:100px !important;
	}
	#settingPage .validate-item .el-input__inner{
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
	#settingPage .el-form-item__error{
		width:200px;
		top:3px;
		left:210px;
	}
	#settingPage .el-icon-sas-warning:before {
		color:#ff4614 !important;
	}
	#settingPage .normalText {
		color:#7a7992;
		font-weight:normal;
		margin-right: 10px;
	}
	#settingPage .navItem .el-icon {
		margin-right: 10px;
	}
	#multiMMEAddPanel .el-form-item__error {
		position: relative;
		left: 0;
	}
	#networkPanel {
		overflow: auto !important;
	}
    
    .activeStatusItem .status-tip,
    .inactiveStatusItem .status-tip {
        display: none;
        position: absolute;
        z-index: 1000;
        padding: 2px 10px;
        background: #fff;
        border-radius: 2px;
        box-shadow: 2px 3px 10px #d1ecf5;
        top: 35px;
    }
    .activeStatusItem:hover .status-tip,
    .inactiveStatusItem:hover .status-tip {
        display: inline-block;
    }
    .mmeIpListBoxCls { 
        overflow:auto;
        margin-bottom:20px;
    }
    .mmeIpListBoxCls .mmeIpItemBoxCls {
        display: inline-block;
        margin-bottom: 8px;
        margin-right: 10px;
    }
    .mmeIpListBoxCls .mmeItemContent {
        display: inline-flex;
        align-items: center;
        border: 1px solid #A0C4F9;
        border-radius: 4px;
        background: #F2F6FF;
        overflow: hidden;
    }
    .mmeIpListBoxCls .mmeItemSpan {
        display: inline-block;
        padding: 0px 10px;
        white-space: nowrap;
        font-size: 12px;
        line-height: 22px;
    }
    .mmeIpListBoxCls .mmeItemSpanFirst {
        border-right: 1px solid #A0C4F9;
    }
    .mmeIpListBoxCls .mmeItemRemove {
        display: inline-block;
        padding: 0 6px;
        cursor: pointer;
        font-size: 14px;
        line-height: 22px;
    }
    .mmeIpListBoxCls .mmeItemRemove:hover {
        color: #F56C6C;
    }
</style>
<div id="settingPage" class="borderPage" style='display:flex;flex-direction:row;flex:1;height:100%;overflow:hidden;'>
	<div class="enbNav">
		<div v-if="showFlagList.CODE_ENB_INFORMATION" class="navItem" :class="currentItem === 'info' ? 'active':''"  @click="changeMain('info')">
			<span class="el-icon el-icon-overview"></span><%=rb.getString("ZongLan")%>
		</div>
		<!--2+4 不显示该菜单-->
		<div v-if="(showFlagList.CODE_ENB_INFORMATION || isNova452) && platformType != 'BM'" class="navItem" :class="currentItem === 'chart' ? 'active':''" @click="changeMain('chart')">
			<span class="el-icon el-icon-operation-statistics"></span><%=rb.getString("TongJi")%>
		</div>
		<!-- BaiBLX-BLQ、BaiBLX-QRTB -->
		<div v-if="isBaiblx_QRTB || isBaiblx_BLQ" class="navItem" :class="currentItem === 'topo' ? 'active':''" @click="changeMain('topo')">
			<span class="el-icon el-icon-operation-topo"></span> TOPO
		</div>
		
		<div v-if="showFlagList.CODE_ENB_SETTINGS" class="navItemGroup CODE_ENB_SETTINGS hidden">
			<div class="navItemTitle"> <span class="el-icon el-icon-operation-settings"></span><%=rb.getString("SheZhi")%></div>
			<div class="navItem" 
				:class="{'active': currentItem === item.code, 'disabled': item.disabled == true}" 
				v-for="item in settingMenus" 
				v-show="item.disabled != true"
				@click="changeSettingTab(item)">
				 {{item.text}}
			</div>
		</div>

		<div v-if="!isSubDevice" class="navItem" :class="currentItem === 'alarm' ? 'active':''" :disabled="isSubDevice" @click="changeMain('alarm')">
			<span class="el-icon el-icon-status-alarm" style='font-size: 18px;'></span><%=rb.getString("GaoJingGuanLi")%>
		</div>
		<div v-if="upgradable && !isSubDevice" class="navItem" :class="currentItem === 'upgrade' ? 'active':''" :disabled="isSubDevice" @click="changeMain('upgrade')">
			<span class="el-icon el-icon-status-kpi-normal" style='font-size: 18px;'></span><%=rb.getString("BanBenGuanLi")%>
		</div>
		<div v-if="!isSubDevice" class="navItem CODE_ENB_BACKUP_RESTORE hidden" :class="currentItem === 'backup' ? 'active':''" :disabled="isSubDevice" @click="changeMain('backup')">
			<span class="el-icon el-icon-circle-backup" style='font-size: 18px;'></span><%=rb.getString("beiFenYuHuiFu")%>
		</div>
		<div v-if="!isSubDevice" class="navItem" :class="currentItem === 'log' ? 'active':''" :disabled="isSubDevice" @click="changeMain('log')">
			<span class="el-icon el-icon-operation-details"></span><%=rb.getString("RiZhiShouJi")%> 
		</div>
		<div v-if="advanceShow" class="navItemGroup advanceGroup">
			<div class="navItemTitle"> <%=rb.getString("GaoJiSheZhi")%></div>
			<div v-if="!isSubDevice" class="navItem" :class="currentItem === 'lic' ? 'active':''" :disabled="isSubDevice" @click="changeMain('lic')">
				<span class="el-icon el-icon-license" style='font-size: 18px;'></span> <%=rb.getString("License")%>
				<i v-if="delay_avaliable=='0' || delay_avaliable=='1'" class="el-icon el-icon-sas-warning" style="margin:0 5px;"></i>
			</div>
			<!-- Cell Scan -->
			<div v-if="isCellScanEnable" class="navItem CODE_ENB_MONITOR hidden" :class="currentItem === 'scan' ? 'active':''" @click="changeMain('scan')">
				<span class="el-icon el-icon-operation-scan"></span> <%=rb.getString("ZhanDianSaoMiao")%>
			</div>
			<div v-if="false" class="navItem" @click="changeMain('cert')">
				<span class="el-icon el-icon-status-validCert"></span> <%=rb.getString("ZhengShu")%>
			</div>
			<div v-if="showFlagList.CODE_ENB_EXPIRY_DATE" class="navItem" :class="currentItem === 'expiry' ? 'active':''" @click="changeMain('expiry')">
				<span class="el-icon el-icon-operation-date"></span> <%=rb.getString("YouXiaoQi")%>
			</div>
			<div v-if="showFlagList.CODE_ENB_TRAFFIC_LIMITATION" class="navItem" :class="currentItem === 'limit' ? 'active':''" @click="changeMain('limit')">
				<span class="el-icon el-icon-operation-limitation"></span> <%=rb.getString("LiuLiangXianZhi")%>
			</div>
			<div v-if="showFlagList.CODE_ENB_DISTRIBUTED && isNotSBS7" class="navItem" :class="currentItem === 'distribute' ? 'active':''" @click="changeMain('distribute')">
				<span class="el-icon-operation-distributed el-icon"></span> <%=rb.getString("FenBuShi")%>
			</div>
			<div v-if="false" class="navItem" @click="changeMain('changePwd')">
				<span class="el-icon el-icon-common-changePassword"></span> <%=rb.getString("XiuGaiMiMa")%>
			</div>
			<div V-if="diagnosticShow" class="navItem" @click="changeMain('diagnostic')">
				<span class="el-icon el-icon-operation-diagnostic"></span> <%=rb.getString("CeSu")%>
			</div>
		</div>
		
	</div>
	<div class="settingMainPage">
		<div class="el-card__header" style="min-height: 40px;">
			<span>{{title}}</span>
			<span class="normalText" style="margin-right:8px;">(<%=rb.getString("XiaoZhanBianMa")%> : <b>{{sn}}</b></span>
			<span class="normalText"><%=rb.getString("HostName")%> : <b>{{name}}</b>)</span>
			<div class="el-icon-common-refresh el-icon" v-if="!['info','chart','topo','alarm','upgrade','backup','log','expiry','limit','distribute','lic','diagnostic'].includes(currentItem)"
			    @click='refreshSettingTab' style="position:absolute;right:50px;font-size:20px;">
			</div>
			<div class="el-icon-common-refresh el-icon" v-else-if="currentItem == 'lic' && writableMap['CODE_ENB_LICENSE'] == true"
				@click='refreshLicenseTab'  style="position:absolute;right:50px;font-size:20px;">
			</div>
			<span class="el-icon el-icon-close" @click="closeSetting"></span>
		</div>
		<div id="setting_main" class="el-card__body" style="position: relative;">
		</div>
		<div v-if="settingOpShow" class="el-card__footer" style="background: #fff;">
			<el-button type="primary" @click="saveSetting" :disabled="submitDisabled"><%=rb.getString("QueDing")%></el-button>
			<el-button @click="closeSetting"><%=rb.getString("QuXiao")%></el-button>
		</div>
	</div>
	
</div>

<script>
var settingVue = new Vue({
	el:'#settingPage',
	data(){
		return {
			submitDisabled: false,

			isBaiblx_QRTB: false,
			isBaiblx_BLQ: false,

			small_cell_code:'',
			status:'',
			sn:'',
			version:"",
			product:"",
			settingMenus: [],
			selectedRow: '',
			settingOpShow: false,
			title:'<%=rb.getString("ZongLan")%>',
			name:'',
			delay_avaliable:'',
			currentItem:'info',
			showFlagList:{
               CODE_ENB_REBOOT:false,
               CODE_ENB_RESET_CONFIG:false,
               CODE_ENB_LOGS:false,
               CODE_ENB_SYNCHRONIZE:false,
               CODE_ENB_HALOB_ENABLE:false,
               CODE_ENB_RF_ENABLE:false,
               CODE_ENB_ACTIVE:false,
               CODE_ENB_SAS_RF_ENABLE:false,
               CODE_ENB_SAS_ENABLE:false,
               CODE_ENB_LOCK:false,
               CODE_ENB_CHANGE_PASSWORD:false,
               CODE_ENB_INFORMATION:false,
               CODE_ENB_SETTINGS:false,
               CODE_ENB_EXPIRY_DATE:false,
               CODE_ENB_EXPIRY_DATE_LOCK:false,
               CODE_ENB_DISTRIBUTED:false,
               CODE_ENB_TRAFFIC_LIMITATION:false,
           },
           curSettingTabData: [],
           isSubDevice:false,

		   isCellScanEnable: false,

		   isNova452: false,

		   platformType: '',
		   
		   source: '', // 记录来源：'monitor' 表示从监控页面跳转
		}
	},
	computed: {
		isNotSBS7() {
			var row = this.selectedRow || {};

			return !['sBS72030','sBS71400'].includes(row.module_type);
		},
		upgradable() {
			return writableMap.CODE_ENB_UPGRADE_FILE || writableMap.CODE_ENB_UPGRADE_FPGA || writableMap.CODE_ENB_UPGRADE_IMAGE || writableMap.CODE_ENB_UPGRADE_PATCH;
		},
		diagnosticShow(){
			return this.version && (this.version.indexOf('BLQ') > -1 || this.version.indexOf('BLN') > -1)
		},
		advanceShow() {
			var vm = this;

			return !vm.isSubDevice || vm.isCellScanEnable || vm.showFlagList.CODE_ENB_EXPIRY_DATE
				|| vm.showFlagList.CODE_ENB_TRAFFIC_LIMITATION || vm.showFlagList.CODE_ENB_DISTRIBUTED
				|| vm.diagnosticShow;
		}
	},
	methods:{
		// 初始化
		init(row,page,source){
			var vm = this,
				code = row.small_cell_code,
				sn = row.serial_number,
				platformType = row.platformType,
				dualCarrierType = row.dual_carrier_type,
				version = row.software_version,
				status = row.connection_status;

			vm.platformType = row.platformType;
			vm.isBaiblx_QRTB = row.platformType == 'BLX' && row.network_model == 'TDDMode';
			vm.isBaiblx_BLQ = row.platformType == 'BLX' && row.network_model == 'FDDMode';
			vm.isCellScanEnable = ['QA_436Q_CA','QA_436Q_SC','QA_436Q_DC','NEU430_DC','BAIBLQ','MLQ'].includes(platformType);

			Render.tableCollector = {};
			vm.small_cell_code = code;
			vm.status = status;
			vm.sn = sn;
			vm.name = row.host_name;
			vm.version = version;
			vm.product = row.product;
			vm.delay_avaliable = row.delay_avaliable;
			vm.source = source || ''; // 记录来源
            vm.selectedRow = row;
			vm.getSettingGroup(row);
			
			var paramsCode={
                    smallCellCode:code
                },
                maintenanceShow = false,
                actionShow = false;

			if(['CR-B4860','QRTB','MLN'].includes(row.product)) {
				vm.isNova452 = true;
			}

            $.ajax({
                type:'POST',
                url:'${ctx}/cell/cpeinfos/getENBOperationItem.action',
                data:paramsCode,
                async:false,
                dataType:'json',
                success:function(data){
                    var operationData = data;
                    if(operationData.Maintenance && operationData.Maintenance.length > 0 ){
                        maintenanceShow = true
                        operationData.Maintenance.map((item)=>{
                            vm.showFlagList[item] = true;
                        });
                    }
                    if(operationData.Actions && operationData.Actions.length > 0 ){
                        actionShow = true
                        operationData.Actions.map((item)=>{
                            vm.showFlagList[item] = true;
                        })
                    }
                    if(operationData.others && operationData.others.length > 0 ){
                        operationData.others.map((item)=>{
                            vm.showFlagList[item] = true;
                        })
                    }
                    
					if(page){
						vm.changeMain(page);
					}else {
						
						if(vm.showFlagList.CODE_ENB_INFORMATION) {
							vm.changeMain('info');
						}else {
							if(vm.isNova452) {
								vm.changeMain('chart');
							}else{
								vm.changeSettingTab(vm.settingMenus[0]);
							}
						}
					}
                },
            });
            
            //辅站 -2设备均屏蔽功能菜单 
            if(row.have_connected == 2 || (['QA_436Q_CA','QA_436Q_SC','QA_436Q_DC','NEU430_DC'].includes(row.platformType) && row.dual_carrier_type == 2) || (['Intel_CR_DC','MLN_DC'].includes(row.platformType) && row.dual_carrier_type == 2)) {// 是否真实可用站
                vm.isSubDevice = true;
            }
			
		},
		changeMain(type){
			var vm = this;
			var code = vm.small_cell_code;
			var urlList = {
				info:'${ctx}/cell/param/toCellDetailParamInfoPage.action?smallCellCode='+code+'&connection_status=' + vm.status + '&timeZone=' + timeZone,
				chart:'${ctx}/enb/setting/openStatisticPage.action',
				topo: '${ctx}/cell/cpeinfos/toCellToPo.action',
				alarm:'${ctx}/enb/setting/openAlarmPage.action',
				upgrade:'${ctx}/enb/setting/openUpgradePage.action',
				backup:'${ctx}/enb/setting/openBackupAndRestorePage.action',
				log:'${ctx}/enb/setting/openLogPage.action',
				lic:'${ctx}/enb/setting/openLicensePage.action',
				cert:'${ctx}/cell/topo/toDashboardTopo.action',
				expiry:'${ctx}/cell/cpeinfos/toCellValidity.action',
				limit:'${ctx}/enb/setting/openOverviewPage.action',
				distribute:'${ctx}/cell/nxp/toCellInfo.action?smallCellCode='+code,
				scan: '${ctx}/cell/cpeinfos/toCellScan.action',
				diagnostic: '${ctx}/enb/setting/openDiagnosticPage.action',
			};
			
			var titleList = {
					info:'<%=rb.getString("ZongLan")%>',
					chart:'<%=rb.getString("TongJi")%>',
					alarm:'<%=rb.getString("GaoJingGuanLi")%>',
					setting:'',
					upgrade:'<%=rb.getString("ShengJi")%>',
					backup:'<%=rb.getString("beiFenYuHuiFu")%>',
					log:'<%=rb.getString("RiZhi")%>',
					lic:'<%=rb.getString("License")%>',
					expiry:'<%=rb.getString("YouXiaoQi")%>',
					limit:'<%=rb.getString("LiuLiangXianZhi")%>',
					distribute:'<%=rb.getString("FenBuShi")%>',
					scan: '<%=rb.getString("ZhanDianSaoMiao")%>',
					diagnostic:'<%=rb.getString("CeSu")%>'
				}
			
			if(vm.isSubDevice && (type == 'alarm' || type == 'upgrade' || type == 'log' || type == 'backup' || type =='lic')){
				return;
			}
			
			vm.settingOpShow = false;
			$('#setting_main').html('');
            $('#setting_main').addClass('loading');
			loadHTML(document.querySelector('#setting_main'),{
                url:urlList[type] ,
                method:'post',
                success: function() {
                	vm.currentItem = type;
                	vm.title = titleList[type];
                	eventBus.$emit("enb-data",vm.selectedRow,vm.small_cell_code,vm.sn,vm.status,vm.version,vm.product,vm.delay_avaliable)
					$('#setting_main').removeClass('loading'); 
                }
            });
		},
		closeSetting(){
			var vm = this;
			// 根据来源判断关闭方式
			if(vm.source === 'monitor'){
				// 从监控页面跳转过来的，触发监控页面的关闭事件
				eventBus.$emit('cancel-enb-setting');
			} else if(vm.source === 'topo') {
				// 从 eNBTopo_tab.jsp 打开的情况
				eventBus.$emit('cancel-enb-topo-setting');
			}
		},
		loadPage(code, id, sn, connStatus){
			var vm = this,
				rowDatas = vm.selectedRow,
				sn = rowDatas.serial_number,
				connectStatus = rowDatas.connection_status,
				platformType = rowDatas.platformType||'',
				rowCode = rowDatas.small_cell_code,
				method = 'get',
				urlList = {
					'Quick Setting' : (platformType && platformType == 'BM') ? '${ctx}/cell/param/toBmQuickSettingPage.action' : '${ctx}/cell/quicksettings/goQuickSettingParamPage.action',
					'Network' : '${ctx}/cell/quicksettings/goNetWorkParamPage.action',
					'LTE' : '${ctx}/cell/quicksettings/goLTEParamPage.action',
					'BTS' : '${ctx}/cell/quicksettings/goBTSParamPage.action',
					'LTE-TURBO': '${ctx}/cell/ap/toAPInfo.action',
					'Special': '${ctx}/cell/quicksettings/goSpecialParamPage.action'
				},
				params = {
					smallCellCode: rowCode,
					enbSerialNumber: sn,
					connectionStatus: connectStatus
				};
			
			if(code == 'LTE-TURBO') {
				method = 'post';
			}
			
			$('#setting_main').html('');
            $('#setting_main').addClass('loading');
			loadHTML(document.querySelector('#setting_main'),{
				url: urlList[code],
				method: method,
				queryParams: params,
				success: function() {
					setTimeout(function(){
						eventBus.$emit('tab-param', rowCode, id, platformType);
					},200);
					
					if(code == 'LTE' || code == 'LTE-TURBO' || code == 'Special') {
						setTimeout(function() {
							$('#setting_main').removeClass('loading');
						}, 500);
					}
				}
			});
		},
		renderData2Dom(data) {
			var vm = this;

            $('#enbSetting_slide_body').html('');
            Render.tableCollector = {};
            Render.gRender(data,document.querySelector('#enbSetting_slide_body'));
            setTimeout(function(){
                $('#enbSetting_slide_body .group-title:not(:first) .title-text').click();
                var group = $("#enbSetting_slide_body .form-group");
                group.each(function(){
                	var item = $(this).find(".form-wrap .form-item");
                	var i =1;
                	item.each(function(){
                		if($(this).hasClass("form-double")){
                			$(this).addClass("double"+i);
                			i++;
                		}
                	})
                })
                
                setTimeout(function(){
                    $('#setting_main').removeClass('loading');
                },500);
            },0);
        },
		settingTabClick(id, code, platform){
			var vm = this;

            // enbPlatform = platform;
			$('#setting_main').html('<form id="enbSetting_slide_body" class="slide-body form-ctn" style="flex:1 auto;overflow: auto;height:calc(100% - 50px);margin:0px;width:auto;padding:0px;background:#fff;"></form>');
            $('#setting_main').addClass('loading');
            $('.form_bt_refresh').hide();

            var postData = {id: id, smallCellCode: code};
            $('#setting_main').data('params',postData);
            $.ajax({
                url: '${ctx}/cell/quicksettings/getParamNodeTreeAndData.action',
                data: postData,
                type: 'post',
                dataType: 'json',
                success: function(data){
                    vm.renderData2Dom(data);
                    setTimeout(function(){
                        $('.form_bt_refresh').show();
                    },500);
                },
                error: function(data){
                    $('#setting_main').removeClass('loading');
                }
            });
        },
		changeSettingTab(mItem) {
			var vm = this,
				rowDatas = vm.selectedRow,
				connectStatus = rowDatas.connection_status,
				platformType = rowDatas.platformType||'',
				rowCode = rowDatas.small_cell_code;
			
			if(mItem.disabled == true) {
				return;
			}

			vm.curSettingTabData = mItem;
			vm.title = mItem.text;
			if(platformType.indexOf('436Q')>=0 || platformType.indexOf('BAIBLQ')>=0 || platformType.indexOf('MLQ')>=0 || platformType.indexOf('BLX')>=0 ){
				vm.settingOpShow = true;
                vm.currentItem = mItem.code;
				if(mItem.code == 'Special'){
					vm.settingOpShow = false;
				}
				vm.loadPage(mItem.code, mItem.id, rowDatas.serial_number, connectStatus);
				return;
			}
			//2+4 = BM 产品类型数据，可进行快速设置
			if(['Quick Setting', '快速设置'].includes(mItem.code) && ['BM'].includes(platformType)){
				vm.settingOpShow = true;
                vm.currentItem = mItem.code;
				vm.loadPage(mItem.code, mItem.id, rowDatas.serial_number, connectStatus);
				return;
			}else{
				vm.settingOpShow = false;
			}

			if(['Network', '网络设置'].includes(mItem.code) && (['BAIBLX'].includes(platformType) || ['Intel_CR_CA','Intel_CR_SC','Intel_CR_TC','Intel_CR_DC','Intel_CR','MLN_SC','MLN_CA','MLN_DC','MLN'].includes(platformType) && rowDatas.dual_carrier_type != 2)){
				vm.settingOpShow = true;
                vm.currentItem = mItem.code;
				vm.loadPage(mItem.code, mItem.id, rowDatas.serial_number, connectStatus);
				return;
			}else{
				vm.settingOpShow = false;
			}
			
			
			if(mItem.text=="LTE-TURBO"){
				window.sessionStorage.setItem('apInfoSn',rowDatas.serial_number)
				var url = '${ctx}/cell/ap/toAPInfo.action'
				var params = {
					smallCellCode:rowDatas.small_cell_code,
					enbSerialNumber:rowDatas.serial_number,
					connectionStatus:rowDatas.connection_status
				}

				$('#setting_main').html('');
				$('#setting_main').addClass('loading');
				$('#setting_main').load(url,params,function(){
					vm.currentItem = mItem.code;
					var ctner = document.querySelector('#setting_main');
					var scripts = ctner.querySelectorAll('script');
					setTimeout(function() {
						Array.from(scripts).map(function(script) { /* 执行远程的脚本 */
							if (ctner.contains(script)) {
								ctner.removeChild(script);
							}
							var newScript = document.createElement('script');
							newScript.type = 'text/javascript';
							newScript.innerHTML = script.innerHTML;
							ctner.appendChild(newScript);
						});
					}, 0);
					
					setTimeout(function() {
						try{
							$.parser.parse(ctner);
						}catch(e){}
					}, 0);
					$('#setting_main').removeClass('loading');
				});
			}else if(["Special","特殊"].includes(mItem.text)) {
				var url = '${ctx}/cell/quicksettings/goSpecialParamPage.action',
					params = {
						smallCellCode: rowDatas.small_cell_code,
						enbSerialNumber: rowDatas.serial_number,
						connectionStatus: rowDatas.connection_status
					};

				$('#setting_main').html('');
				$('#setting_main').addClass('loading');
				$('#setting_main').load(url,params,function(){
					vm.currentItem = mItem.code;
					$('#setting_main').removeClass('loading');
				});
				$("#setting_main").data('params',{id:mItem.code,smallCellCode:rowDatas.small_cell_code});
			}else{
				var params = Render.getFormDatas($('#setting_main')),
					edit = !isEmptyJson(params),
					addEdit = false;

                vm.currentItem = mItem.code;
				if(addEdit){
					//turnTabs(navItem);
					vm.settingTabClick(mItem.id, rowCode, mItem.platform);
				}else{
					if(edit){// 参数有变动时，确认提示
						$.messager.confirm('Confirm','<%=rb.getString("QueDingLiKaiDangQianYeMian")%>',function(r){
							if(r){
								//turnTabs(navItem);
								vm.settingTabClick(mItem.id, rowCode, mItem.platform);
							}
						});
					}else{// 无变动直接跳转
						//turnTabs(navItem);
						vm.settingTabClick(mItem.id, rowCode, mItem.platform);
					}
				}
			
			}
		},
		getSettingGroup(row) {
			var vm = this,
				connectStatus = row.connection_status,
                cellCode = row.small_cell_code,
				platformType = row.platformType||'',
				dualCarrierType = row.dual_carrier_type;
			
			//判断菜单的位置
			$.ajax({
				url: '${ctx}/cell/quicksettings/getSettingGroupTree.action',
				dataType: 'json',
				async: false,
				type: 'post',
				data: {
					title:'Settings',
					smallCellCode: cellCode
				},
				success: function(json){
					if(json) {
						vm.settingMenus = json;
						if(platformType.indexOf('436Q')>=0 || platformType.indexOf('BLQ')>=0 || platformType.indexOf('BLX')>=0 || platformType.indexOf('MLQ')>=0){

							var disabled = connectStatus == 'Off' || dualCarrierType == '2';

							vm.settingMenus.map(function(item){
								item.show = true;
								item.disabled = disabled;
								
								if(item.code == 'LTE-TURBO') {
									item.show = isLWAEnable;
								}
								
								if(item.code == 'Quick Setting' || item.code == "Basic") {
									item.disabled = false;
								}

								if(item.code == 'Special' && vm.isBaiblx_BLQ) {
									item.disabled = true;
								}
									
								//qrtb 辅小区 不可点击 LTE  -- 10.2.4
								//#73792 10.3.0 放开 LTE 设置
								if(['QA_436Q_DC'].includes(platformType) && dualCarrierType == 2 && ['LTE','LTE设置'].includes(item.text)) {
									item.disabled = false;
								}
							});
							
							//vm.activeCode = res.data[0].code;
							//vm.loadPage(json[0].code, json[0].id, row.serial_number, connectStatus);
						}else {
							vm.settingMenus.map(function(item){
								if(item.code == "Basic"){
									item.disabled = false;
								}else{
									if(connectStatus == 'Off') item.disabled = true;
									
									if(['QA_436Q_DC','NEU430_DC','Intel_CR_DC','MLN_DC'].includes(platformType) && dualCarrierType == 2) { // 新类型QA_436Q_DC的处理，辅波只有basic可以设置
										
										if(['QA_436Q_DC','Intel_CR_DC','MLN_DC'].includes(platformType) && ['LTE','LTE设置'].includes(item.text)) {// 4860 LTE 放开设置 调整QA_436Q_DC 放开LTE设置
											
										}else {
											item.disabled = true;
										}
									}
								}
							});
						}
					}
				}
			});
		},
		//license Tab 右上角刷新
		refreshLicenseTab(){
			var vm = this,
				params = {
					smallCellCode: vm.small_cell_code
				};
			
			// 发起同步指令- 刷新页面内容,分两种情况
			axios.post("${ctx}/cell/quicksettings/syncLicense.action",stringify(params)).then(function(res){
				var data = res.data;
				if(data["success"]){
					//根据 getMsg.action 推送的消息进行数据刷新
					vm.changeMain('lic');
				}else{
					vm.$message.error(data["message"])
				}
			})
		},
		//右上角刷新功能
		refreshSettingTab(){
			var vm= this,
				params = {
					id: vm.curSettingTabData.id,
					smallCellCode: vm.small_cell_code
				};
			
			// 发起同步指令- 刷新页面内容,分两种情况
			axios.post("${ctx}/cell/quicksettings/sync.action",stringify(params)).then(function(res){
				var data = res.data;
				if(data["success"]){
					var mItem = vm.settingMenus.filter(function(item){
							return item.id == vm.curSettingTabData.id;
						});

					if(mItem.length) vm.changeSettingTab(mItem[0]);
					
					/*
					//分条件映射到页面
					var platformType = vm.selectedRow.platformType||''; 
					if(platformType.indexOf('436Q')>=0 || platformType.indexOf('BAIBLQ')>=0 || platformType.indexOf('MLQ')>=0){
						vm.settingOpShow = true;
						vm.loadPage(vm.curSettingTabData.code, vm.curSettingTabData.id, vm.selectedRow.serial_number, vm.selectedRow.connection_status);
						return;
					}else{
						//vm.settingTabClick(mItem.id, rowCode, mItem.platform);
						vm.settingOpShow = false;
						vm.settingTabClick(vm.curSettingTabData.id, vm.small_cell_code, vm.selectedRow.platform_flag);
					}
					*/
				}
			})
		},
		saveSetting(){
			eventBus.$emit('save-set')
		},
	},
	mounted(){
		eventBus.$off("row-data").$on("row-data",this.init)
	}
})
</script>

<script>
    var Kai = '<%=rb.getString("Kai")%>',
        Guan = '<%=rb.getString("Guan")%>',
        yilianjie = '<%=rb.getString("LianJieZhengChang")%>',
        weilainjie = '<%=rb.getString("LianJieDuanKai")%>';
    // 基站设置国际化
	$.renderDefaultOptions.system = '${ctx}'?'${ctx}/':'';
	Render.submitTxt = '<%=rb.getString("QueDing")%>';
	Render.resetTxt = '<%=rb.getString("QuXiao")%>';
	Render.propOkTxt = '<%=rb.getString("QueDing")%>';
	Render.propCancelTxt = '<%=rb.getString("QuXiao")%>';
	var TISHI = '<%=rb.getString("TiShi")%>',
		CHENGGONG = '<%=rb.getString("ChengGong")%>';
		
	Render.status.success = CHENGGONG;
	Render.status.operation = '<%=rb.getString("CaoZuo")%>';
	Render.status.tips = '<%=rb.getString("QueRen")%>';
	Render.status.confirm = '<%=rb.getString("QueRenCheXiao")%>';
	Render.status.remove = '<%=rb.getString("QueRenShanChu")%>';
	var rebootText= '<%=rb.getString("SheZhiHouChongQi")%>',	
		closeTrue = true,
		addEdit = false,
		settingTimeout = null;

	// 基站设置提交
	Render.submit = function(){
		var valid = Render.valid();
		if(!valid) return;
		
		var params = Render.getFormDatas($('#enbSetting_slide_body'));
		// 判定是否有修改项，无则提示且不提交
		if(isEmptyJson(params)) {
			showMsg('prompt_msg','<%=rb.getString("CanShuZhiMeiYouBianHua")%>');
			return;
		}
		
		// 在判定是否有重启项之前，先关闭所有展开的下拉框
		hideAllComboBoxPanels();
		
		// 判定是否有重启项
		if(Render.validResult.reboot){
			var tipContent = [
					"<%=rb.getString("JiZhanChongQiTiShi")%>",
					"<br/><br/>",
					"<input id='reboot_confirm_status' type='checkbox' />",
					" <label for='reboot_confirm_status' style='font-size: 14px;color: #1DA3FC;cursor: pointer;'><%=rb.getString("SheZhiHouChongQi")%></label>"
				].join("");
			var msger = $.messager.confirm("<%=rb.getString("QueRen")%>", tipContent, function (r) {
				if (r) {
					/* 重启勾选判断 */
					var needReboot = false,
						rebootCkbox = $('#reboot_confirm_status',msger);
					if(rebootCkbox.length && rebootCkbox.prop('checked')){
						needReboot = true;
					}
					msger = null;
					
					$('#setting_main').addClass('loading');
					setSettingTimeout();
					var rowCode = settingVue.selectedRow.small_cell_code;
					$.post('${ctx}/cell/quicksettings/saveParamValue.action?smallCellCode='+rowCode, {"params": JSON.stringify(params)}, function(data){
						if(data.success){
							showMsg('success_msg',"<%=rb.getString("ChengGong")%>");
						
							// 勾选重启，下发重启指令
							if(needReboot) {
								$.post("${ctx}/cell/cpeinfos/cellReboot.action", {cell_code: rowCode}, function (data) {
									if (!data["success"]) {
										showMsg('error_msg',data["message"]);
									}
								}, "json");
							}
							addEdit = true;
							closeSettingPanel();
						}else{
							// 设置失败相关提示
							if(data.validMsg){
								$.messager.alert(TISHI,data.validMsg,'warning');
							}else{
								$.messager.confirm(TISHI,'<%=rb.getString("JiZhanSheZhiShiBai")%>',function(r){
									if(r) closeSettingPanel();
								});
							}
							addEdit = false;
						}
						Render.validResult.reboot = false;
						$('#setting_main').removeClass('loading');
						resetSettingTimeout();
					},'json');
				}
			}).addClass("seriousConfirm");
		}else{
			$('#setting_main').addClass('loading');
			setSettingTimeout();
			var rowCode = settingVue.small_cell_code;
			$.post('${ctx}/cell/quicksettings/saveParamValue.action?smallCellCode='+rowCode, {params: JSON.stringify(params)}, function(data){
				if(data.success){
					// $('.form-operations .success').addClass('show');
					showMsg("success_msg",'<%=rb.getString("ChengGong")%>')
					closeSettingPanel();
					if(Render.validResult.reboot) {
						$.post("${ctx}/cell/cpeinfos/cellReboot.action", {cell_code: rowCode}, function (data) {
							if (!data["success"]) {
								showMsg('error_msg',data["message"]);
							}
						}, "json");
					}
						addEdit = true
				}else{
					if(data.validMsg){
						$.messager.alert(TISHI,data.validMsg,'warning');
					}else{
						$.messager.confirm(TISHI,'<%=rb.getString("JiZhanSheZhiShiBai")%>',function(r){
							if(r) closeSettingPanel();;
						});
					}
					addEdit = false
				}
				
				Render.validResult.reboot = false;
				$('#setting_main').removeClass('loading');
				resetSettingTimeout();
			},'json');
		}
	}
	// 基站设置页取消
	Render.reset = function(){
		var params = Render.getFormDatas($('#enbSetting_slide_body')),toClose=false;
		var edit = !isEmptyJson(params)
		if(edit){
			if(!addEdit){
				closeTrue =true
			}else{
				closeTrue =false
			}
		}else{
			closeTrue = true
		}
		closeSettingPanel(closeTrue);
	}

	function closeSettingPanel(bool) {
		var addEdit = false
		var params = Render.getFormDatas($('#enbSetting_slide_body')),toClose=false;
		if(bool==true){
			if(addEdit){ // 保存后变为true 在点击关闭或者取消就会直接关闭
				settingVue.closeSetting();
			}else{ // 没有点击保存 为false 
				if(!isEmptyJson(params)) {
					$.messager.confirm(TISHI,'<%=rb.getString("QueDingLiKaiDangQianYeMian")%>',function(r){
						if(r) settingVue.closeSetting();
					}).addClass("seriousConfirm");
				}else{
					settingVue.closeSetting();
				}
			}
		}else{
			settingVue.closeSetting();
		}
	}
	// 初始化快速设置同步事件
	$('#setting_main .form_bt_refresh').off('click').on('click',function(){
		var slider = $('#enbSetting_slide'),
			sliderForm = $('#setting_form_cnt'),
			postData = slider.data('params');
		// 设置等待蒙层
		sliderForm.addClass('loading');
		// 发起同步指令
		$.ajax({
			url: '${ctx}/cell/quicksettings/sync.action',
			data: postData,
			type: 'post',
			dataType: 'json',
			success: function(data){
				// 取消等待蒙层
				sliderForm.removeClass('loading');
			},
			error: function(data){
				sliderForm.removeClass('loading');
			}
		});
	});
	// 基站设置输入域的校验提示国际化
	Render.messages = function(opts){
		var value = opts.value || '';

		if(opts.type=='select'){
			var sDom = $('#'+opts.name,$('.form-ctn'));
			if(opts.typeFlag == 'ipsec'){
				sDom = $('[comboname="'+opts.name+'"]',$('.form-ctn'));
			}
			var datas = sDom.combobox('getData');
			if(datas){
				datas.map(function(row){
					if(row.value==opts.value) value = row.text;
				});
			}
		}

		var msges = {
				valid: '',
				required: '<%=rb.getString("BiTian")%>',
				range:'<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%>' + opts.min + '-' + opts.max + '<%=rb.getString("JuHao")%><%=rb.getString("ZhengXing")%>',
				length: '<%=rb.getString("ChangDuJiaoYan")%>' + opts.minlength + '-' + opts.maxlength,
				reboot: '<%=rb.getString("ChongQiJiaoYan")%>' + value
			}
		
		return msges;
	}
	// 基站设置输入域blur事件触发的校验方法
	var eNbSetting = {
		/**
		* ipsec表格数据处理
		* @param data{object}: 表格数据
		**/
		nlListLoadFilter: function(data){
			data.rows = data.rows.map(function(row, index){
				if(index == 0) row._remove = false;
				
				return row;
			});

			return data;
		},
		ipsecTbChange(newVal,oldVal,props) {
			var id = '3EB80FC400901B4B6C0EA0E0B0D24EB1',
				key = '71F8B55F13A428E19105F8F916E885CC',
				sctn = $('#'+id),
				arr = (newVal||[]).map(function(row){
					return row[key];
				}),
				sufStr = ':LTE_X_BAICELLS_POOL_MME_LIST',
				list = [],
				data = [];

			if(arr.length) {
				arr.unshift('');

				arr.map(function(item){
					arr.map(function(item2){
						if(item) {
							if(item != item2) {
								list.push(item+sufStr+'1' + (item2?','+item2+sufStr+'2':''));
							}
						}else {
							if(item2) {
								list.push(item2+sufStr+'2');
							}
						}
					})
				});

				data = list.map(function(m){
					return {
						text: m,
						value: m
					};
				});
			}

			if(key) {
				sctn.combobox('loadData', data);
				//sctn.combobox('setValue', '');
			}
		},
		/**
		* 天线切换
		* @param newVal{string}: 当前值
		* @param oldVal{string}: 历史值
		* @param props{object}: 属性集合
		**/
		antennaChange(newVal,oldVal,props) {
			var id = '33861BCB5CBDA9E14AA54A82ADF9000A',
				mult = $('#'+id),
				props = mult.data('props'),
				isHidden = props.hidden == '1';

			if(!isHidden) { // 可见时才触发联动
				if(newVal == '4' && mult.length) {
					if(mult.attr('oldvalue') !== mult.combobox('getValue') || $('#'+props.name).attr('oldvalue') !== '4') {
						mult.combobox('setValue','1');
					}
					mult.combobox('readonly',true);
				}else if(mult.length) {
					mult.combobox('readonly',false);
				}
			}
		},
		antenna2Change(newVal,oldVal,props) {
			var id = '33861BCB5CBDA9E14AA54A82ADF9000B',
				mult = $('#'+id),
				props = mult.data('props'),
				isHidden = props.hidden == '1';

			if(!isHidden) { // 可见时才触发联动
				if(newVal == '4' && mult.length) {
					if(mult.attr('oldvalue') !== mult.combobox('getValue') || $('#'+props.name).attr('oldvalue') !== '4') {
						mult.combobox('setValue','1');
					}
					mult.combobox('readonly',true);
				}else if(mult.length) {
					mult.combobox('readonly',false);
				}
			}
		},
		antenna3Change(newVal,oldVal,props) {
			var id = '33861BCB5CBDA9E14AA54A82ADF9000C',
				mult = $('#'+id),
				props = mult.data('props'),
				isHidden = props.hidden == '1';

			if(!isHidden) { // 可见时才触发联动
				if(newVal == '4' && mult.length) {
					if(mult.attr('oldvalue') !== mult.combobox('getValue') || $('#'+props.name).attr('oldvalue') !== '4') {
						mult.combobox('setValue','1');
					}
					mult.combobox('readonly',true);
				}else if(mult.length) {
					mult.combobox('readonly',false);
				}
			}
		},
		multipleChange(newVal,oldVal,props) {
			var id = '33861BCB5CBDA9E14AA54A82ADFB71F1',
				mult = $('#'+id),
				rangeVal = mult.combobox('getValue'),
				list43 = [],
				list46 = [];

			if(props.hidden == 1) return;

			for(var i=37; i<=43; i++) {
				list43.push({
					text: i + 'dBm',
					value: i + ''
				})
				list46.push({
					text: i + 'dBm',
					value: i + ''
				})
			}
			list46.push({
				text: '44dBm',
				value: '44'
			})
			list46.push({
				text: '45dBm',
				value: '45'
			})
			list46.push({
				text: '46dBm',
				value: '46'
			})

			if(newVal == '1' && mult.length) {
				mult.combobox('loadData',list43);
				if(rangeVal > 43) mult.combobox('setValue','43');
			}else if(mult.length) {
				mult.combobox('loadData',list46);
			}
		},
		// 核对  RTD  RTS  这三个字段的id  以及辅小区都是 相同 id
		//频段 Band： '2471297B9C7BC4137689554491451DF5'
		//带宽 BandWidth-RTS： '12384CEFA54AE5C42C44671754610462'
		//频点 Earfcn：'6D8BE1BA5904A862B9D87E85BE1E8455' 
		// Intel： RTS RTD 涉及频段相关逻辑
		newBandChange(newVal,oldVal,props) {
			var	bandwidthId = $('#'+'12384CEFA54AE5C42C44671754610462'),
				bandwidthVal = bandwidthId.combobox('getValue'); // 带宽的值
			//调用此方法，映射频点  传递频段，带宽的值
			if(newVal && bandwidthVal){
				eNbSetting.commonChangeEarfcn(newVal,bandwidthVal);
			}
		},
		// Intel： RTS RTD 涉及带宽相关逻辑
		newBandWidthChange(newVal,oldVal,props) {
			var	bandId = $('#'+'2471297B9C7BC4137689554491451DF5'),
				bandVal = bandId.combobox('getValue'); // 频段的值
			//调用此方法，映射频点  传递频段，带宽的值
			if(bandVal && newVal){
				eNbSetting.commonChangeEarfcn(bandVal,newVal);
			}
		},
		commonChangeEarfcn: function(newBandVal, newBandWidthVal){
			var id = '6D8BE1BA5904A862B9D87E85BE1E8455', //频点的 id
				$tar = $('#'+id),
				earfcnVal = $tar.combobox('getValue'), // 频点的值
				bandMap = [],
				iVal = '', // 计算的循环数
				minEarfcnRange = '', //最小范围
				maxEarfcnRange = '', //最大范围
				stepLeader = 5, // 步长
				params = {},
				curModuleType = settingVue.selectedRow.module_type,
				curPlatformType = settingVue.selectedRow.platformType;

			if(curModuleType !='' && curPlatformType !='' && curModuleType !=undefined && curPlatformType !=undefined && curModuleType !=null && curPlatformType !=null){
				params.module_type = curModuleType;
				params.platformType = curPlatformType;			
			
				axios.post("${ctx}/cell/quicksettings/getEarfcnByPlatformTypeAndModuleType.action",stringify(params)).then(function(res){
					var data = res.data;
					
					if(data.length > 0){
						data.map(function(item){
							//根据当前已选带宽及频段，映射匹配的数据
							if(item.bandwidth == newBandWidthVal && item.band == newBandVal){
								minEarfcnRange = Number(item.min_earfcn);
								maxEarfcnRange = Number(item.max_earfcn);
								//带宽 5, 15： 步长为5； 带宽10, 20： 步长为10
								
								if(item.bandwidth == 'n25' || item.bandwidth == 'n75'){
									//步长 5 
									stepLeader = 5;
								}else{
									//步长 10
									stepLeader = 10;
								}
								
								iVal= (maxEarfcnRange - minEarfcnRange) / stepLeader;
							}
						});
						
						for(var i=0; i<=iVal; i++) bandMap.push({ text: earfcnFormatter(minEarfcnRange + i*stepLeader + ''), value: minEarfcnRange + i*stepLeader + '' });
						//频段 带宽的值 
						if($tar.length && newBandVal && newBandWidthVal) {
							$tar.combobox('loadData', bandMap);
							if(earfcnVal != $tar.combobox('options').originalValue) {
								$tar.combobox('setValue', '');
							}
						}
					}
				})
			}
		},		
		bandWidthChange(newVal,oldVal,props) {
			var id = '6D8BE1BA5904A862B9D87E85BE1E8455',
				$tar = $('#'+id),
				earfcnVal = $tar.combobox('getValue'),
				bandMap = {
					'n25': [], // 39675 -- 41565 : step 5 -- 378   global: 36225 -- 65375
					'n50': [], // 39700 -- 41540 : step 10 -- 184  global: 36250 -- 65350
					'n75': [], // 39725 -- 41515 : step 5 -- 358   global: 36275 -- 65325
					'n100': [],// 39750 -- 41490 : step 10 -- 174  global: 37850 -- 65300
				};
			
			// 40961
			/*
			for(var i=0; i<=378; i++) bandMap['n25'].push({ text: earfcnFormatter(39675 + i*5 + ''), value: 39675 + i*5 + '' });
			for(var i=0; i<=184; i++) bandMap['n50'].push({ text: earfcnFormatter(39700 + i*10 + ''), value: 39700 + i*10 + '' });
			for(var i=0; i<=358; i++) bandMap['n75'].push({ text: earfcnFormatter(39725 + i*5 + ''), value: 39725 + i*5 + '' });
			for(var i=0; i<=174; i++) bandMap['n100'].push({ text: earfcnFormatter(39750 + i*10 + ''), value: 39750 + i*10 + '' });
			bandMap['n25'].push({ text: '40961(2627.1MHz)', value: '40961' });
			bandMap['n50'].push({ text: '40961(2627.1MHz)', value: '40961' });
			bandMap['n75'].push({ text: '40961(2627.1MHz)', value: '40961' });
			bandMap['n100'].push({ text: '40961(2627.1MHz)', value: '40961' });
			*/
			for(var i=0; i<=5830; i++) bandMap['n25'].push({ text: earfcnFormatter(36225 + i*5 + ''), value: 36225 + i*5 + '' });
			for(var i=0; i<=2910; i++) bandMap['n50'].push({ text: earfcnFormatter(36250 + i*10 + ''), value: 36250 + i*10 + '' });
			for(var i=0; i<=5810; i++) bandMap['n75'].push({ text: earfcnFormatter(36275 + i*5 + ''), value: 36275 + i*5 + '' });
			for(var i=0; i<=2745; i++) bandMap['n100'].push({ text: earfcnFormatter(37850 + i*10 + ''), value: 37850 + i*10 + '' });

			if($tar.length && newVal) {
				var valList = bandMap[newVal].map(function(item){
						return item.value
					});
				
				$tar.combobox('loadData', bandMap[newVal]);
				if(earfcnVal != $tar.combobox('options').originalValue) {
					//$tar.combobox('setValue', '40961');
					$tar.combobox('setValue', '');
				}
			}
		},
		bandWidthRTDChange(newVal,oldVal,props) {
			var id = '6D8BE1BA5904A862B9D87E85BE1E8455',
				$tar = $('#'+id),
				earfcnVal = $tar.combobox('getValue'),
				bandMap = {
					'n25': [], // 38675 -- 39625 : step 5 -- 190
					'n50': [], // 38700 -- 39600 : step 10 -- 90
					'n75': [], // 38725 -- 39575 : step 5 -- 170
					'n100': [],// 38750 -- 39550 : step 10 -- 80
				};
			
			for(var i=0; i<=190; i++) bandMap['n25'].push({ text: earfcnFormatter(38675 + i*5 + ''), value: 38675 + i*5 + '' });
			for(var i=0; i<=90; i++) bandMap['n50'].push({ text: earfcnFormatter(38700 + i*10 + ''), value: 38700 + i*10 + '' });
			for(var i=0; i<=170; i++) bandMap['n75'].push({ text: earfcnFormatter(38725 + i*5 + ''), value: 38725 + i*5 + '' });
			for(var i=0; i<=80; i++) bandMap['n100'].push({ text: earfcnFormatter(38750 + i*10 + ''), value: 38750 + i*10 + '' });

			if($tar.length && newVal) {
				$tar.combobox('loadData', bandMap[newVal]);
				if(earfcnVal != $tar.combobox('options').originalValue) {
					$tar.combobox('setValue', '');
				}
			}
		},
		/**
		* 开关切换
		* @param newVal{string}: 当前值
		* @param oldVal{string}: 历史值
		* @param props{object}: 属性集合
		**/
		switchChange: function(newVal,oldVal,props){
			eNbSetting.resetIp(newVal,oldVal,props);
			/* 级联联动关系 */
			if(props.cascade) {
				var cascade = eval('('+props.cascade+')');
				cascade.map(function(item){
					// 级联显示项
					if(item.show && newVal == item.value){
						item.show.map(function(id){
							var ctn = $('#'+id),
							pros = ctn.data('props');
							if(pros.type=='select'){
								var origVal = ctn.combobox('getValue');
								ctn.combobox('setValue','').combobox('setValue',origVal)
							}
						});
					}
					// 级联隐藏项
					if(item.hide && newVal == item.value){
						item.hide.map(function(id){
							setTimeout(function(){
								$('#'+id).parents('.form-item').addClass('form-hidden');
							},10);
						});
					}
				})
			}
		},
		/**
		* 模式切换
		* @param newVal{string}: 当前值
		* @param oldVal{string}: 历史值
		* @param props{object}: 属性集合
		**/
		modleChange: function(newVal,oldVal,props){
			eNbSetting.resetIp(newVal,oldVal,props);
			/* 级联关系 */
			if(props.cascade) {
				var cascade = eval('('+props.cascade+')');
				cascade.map(function(item){
					if(item.show && newVal == item.value){
						item.show.map(function(id){
							var ctn = $('#'+id),
								pros = ctn.data('props');

							if(pros.type=='select'){
								var origVal = ctn.combobox('getValue');
								ctn.combobox('setValue','').combobox('setValue',origVal)
							}
						});
					}
				})
			}
		},
		enbModelChange: function(newVal,oldVal,props){
			/* 级联关系 */
			if(props.cascade) {
				var cascade = eval('('+props.cascade+')');
				cascade.map(function(item){
					if(item.show && newVal == item.value){
						item.show.map(function(id){
							var ctn = $('#'+id),
								pros = ctn.data('props');

							if(pros.type=='select'){
								var origVal = ctn.combobox('getValue');
								ctn.combobox('setValue','').combobox('setValue',origVal)
							}
						});
					}
				})
			}
		},
		/**
		* 表格数据加载前事件
		* @param param{object}: 查询参数
		**/
		onBeforeLoad: function(param){
			var rowCode = settingVue.selectedRow.small_cell_code;
			param['smallCellCode'] = rowCode;
			sessionStorage.setItem('smallCellCode',rowCode)
		},
		/**
		* ipsec添加事件
		* @param opts{object}: 属性集合
		**/
		ipsecAdd: function(opts){
			var tb = $('#F3965F1B73440F98C800591D56848CD9');
			if($('#66C118DE22B6A1146D3A1A5FF7B66BC3').length) tb = $('#66C118DE22B6A1146D3A1A5FF7B66BC3');
			if($('#EF12ED11C7A3167C15CCF78EF40ABF5C').length) tb = $('#EF12ED11C7A3167C15CCF78EF40ABF5C');
			if($('#B20E15F6543059F57AE7476FE0679C81').length) tb = $('#B20E15F6543059F57AE7476FE0679C81');
			if($('#0D3FD79C1A7FACBDA596A3F7E2B269DA').length) tb = $('#0D3FD79C1A7FACBDA596A3F7E2B269DA');
			if($('#54753447B54A61ACC314A98B4FB507B1').length) tb = $('#54753447B54A61ACC314A98B4FB507B1');
			if($('#54C26FB9A9A8E590156CA1AE142347E6').length) tb = $('#54C26FB9A9A8E590156CA1AE142347E6');
			
			tb.datagrid('unselectAll');
			// 添加的数据不超过2条
			var rows = tb.datagrid('getRows');
			if(rows.length>=2) {
				return ;
			}
			
			var form = $('<form id="ipsecForm" class="flex-ctn" operateType="add" style="height: 100%;"></form>');
			$('.form-ctn .form-props').html('').append(form);
			// 加载ipsec的jsp片段
			form.load($.renderDefaultOptions.system+opts.propsUrl,function(data){
				$.parser.parse(this);
				openPropsPanel();
				eNbSetting.changeTitle(opts.label);
			});
		},
		/**
		* ipsec修改事件
		* @param opts{object}: 属性集合
		* @param row{object}: 表格行数据
		**/
		ipsecEdit: function(opts,row){
			var form = $('<form id="ipsecForm" class="flex-ctn" operateType="edit" style="height: 100%;"></form>');
			$('.form-ctn .form-props').html('').append(form);
			// 加载ipsec的jsp片段
			form.load($.renderDefaultOptions.system+opts.propsUrl,function(data){
				$.parser.parse(this);
				openPropsPanel();
				eNbSetting.changeTitle(opts.label);
			});
		},
		/**
		* 标题切换
		* @param title{string}: 标题文本
		**/
		changeTitle: function(title){
			var tdom = $('#enbSetting_slide .slidebarTitleContainer .default');
			tText = tdom.html();

			if(tdom.data('old')){
				
			}else{
				tdom.data('old',tText);
			}

			if(title == false) tdom.html(tdom.data('old'));
			else tdom.html(title);
		},
		/**
		* ipsec表格数据处理
		* @param data{object}: 表格数据
		**/
		ipsecLoadFilter_qc: function(data){
			data.rows = data.rows.map(function(row){
				row._remove = false;
				return row;
			});

			return data;
		},
		/**
		* mme ip添加校验
		* @param props{object}: 属性集合
		**/
		mmeIpClick: function(props){
			var value = $('#'+props.name+'_show').textbox('getValue');

			var ctner = $('#'+props.name+'_show').parents('.form-item');
			// ip校验通过设值
			if(isValidIP(value)){
				var origVal = $('#'+props.name).textbox('getValue');
				if(origVal){
					var list = origVal.split(','),
						existed = false;
					
					if(list.includes(value.trim())) {
						existed = true;
					}

					if(list.length>=16) {
						ctner.attr('data-msg','max: 16').addClass('invalid');
						setTimeout(function(){
							$('#'+props.name).next().find('input').blur();
						},3000);
						
						return;
					}
					
					if(existed) return;
					else origVal += ','+value;
				}else{
					origVal = value;
				}

				$('#'+props.name).textbox('setValue',origVal);
				$('#'+props.name+'_show').textbox('setValue','');
				$('#'+props.name).next().find('input').blur();
			}
			if(value == ''){
				ctner.attr('data-msg','<%=rb.getString("IPShuRuTiShi")%>').addClass('invalid');
				setTimeout(function(){
					$('#'+props.name).next().find('input').blur();
				},3000)
			}
		},
		/**
		* 校验ip
		* @param e{event}: 鼠标事件
		* @param props{object}: 属性集合
		**/
		ipValid: function(e,props){
			var value = $('#'+props.name).textbox('getValue'),
				nowVal = $('#'+props.name.replace('_show','')).textbox('getValue'),
				originVal = $('#'+props.name.replace('_show','')).textbox('options').originalValue;
			
			var ctner = $('#'+props.name).parents('.form-item');
			// ip 是否重复
			if(isValidIP(value)){
				var list = nowVal.split(','),
					existed = false;
				
				list.map(function(ipItem){
					if(ipItem.indexOf(value.trim())>=0) existed = true;
				});

				if(existed && props.button) return '<%=rb.getString("YiCunZai")%>';
				else {
					if(props.button) $('#'+props.name.replace('_show','')).next().find('input').blur();
					return '';
				}
			}else if(value != '' && value!=originVal){
				return '<%=rb.getString("IPDiZhiFeiFa")%>';
			}else{
				if(props.button) $('#'+props.name.replace('_show','')).next().find('input').blur();
			}
		},
		/**
		* ip值改变事件
		* @param val{string}: 当前值
		* @param oldVal{string}: 历史值
		* @param props{object}: 属性集合
		**/
		ipChange: function(val,oldVal,props){
			var ctner = $('#'+props.name).parents('.form-item');
			$('.flex-ctn-row',ctner).remove();

			if(val){
				var arr = val.split(',');
				arr.map(function(item){
					if(item) eNbSetting.addSuffix(item,ctner);
				});
			}
		},
		/**
		* 添加后缀
		* @param val{string}: 当前值
		* @param ctner{dom}: dom容器节点
		**/
		addSuffix: function(val,ctner){
			var sufCtn = $('.flex-ctn-row',ctner);
			if(sufCtn.length==0){
				sufCtn = $('<div class="flex-ctn-row" style="padding-left: 3px;"></div>');
				ctner.append(sufCtn);
			}

			var suffix = $('<div class="form-suffix"><span class="text">'+val+'</span> <span class="form-bt-remove el-icon el-icon-operation-delete"></span></div>');
			// 删除绑定事件
			suffix.find('.form-bt-remove').on('click',function(){
				suffix.remove();
				var tb = ctner.find('input[textboxname]:first'), ipval = '';
				
				$('.flex-ctn-row>.form-suffix',ctner).each(function(n,item){
					if(ipval) ipval += ',';
					ipval += $(item).find('.text').text();
				});
				tb.textbox('setValue',ipval);
				tb.next().find('input').blur();
			});

			sufCtn.append(suffix);
		},
		rootIndexBlur: function(e,props) {
			var val = $('#'+props.name).textbox('getValue')||'',
				regRange = /^(\d+)..(\d+)$/,
				valid = true,
				regInt = /^\d+$/,
				min = props.min-0 || 0,
				max = props.max-0 || 837;
			
			if(regRange.test(val)){
				// ..分隔多值
				var m = val.match(regRange),
					first = m[1],second=m[2];
				if(first<second){
					if(first<min || first>max) valid = false;
					if(second<min || second>max) valid = false;
				}else valid = false;
			}else{
				// 单值
				if(isNaN(val) || val<min || val>max || !regInt.test(val)) valid = false;
			}
			
			if(valid) return '';
			else {
				return 'Int,or range like min..max ['+min+'-'+max+']';
			}
		},
		/**
		* pci校验
		* @param e{event}: 鼠标事件
		* @param props{object}: 属性集合
		**/
		pciBlur: function(e,props){
			var val = $('#'+props.name).textbox('getValue')||'',
				reg = /^\d(,\d)*$/,
				regRange = /^(\d+)..(\d+)$/,
				valid = true,
				regInt = /^-?\d+$/,
				min = props.min-0 || 0,
				max = props.max-0 || 503;
			
			if(val.indexOf(',')>=0){
				// 逗号分隔多值
				var arr = val.split(',');
				arr.map(function(value){
					if(isNaN(value) || value<min || value>max) valid = false;
				});
			}else if(regRange.test(val)){
				// ..分隔多值
				var m = val.match(regRange),
					first = m[1],second=m[2];
				if(first<second){
					if(first<min || first>max) valid = false;
					if(second<min || second>max) valid = false;
				}else valid = false;
			}else{
				// 单值
				if(isNaN(val) || val<min || val>max || !regInt.test(val)) valid = false;
			}
			
			if(valid) return '';
			else {
				return 'PCI list separated by semicolons,or range like min..max ['+min+'-'+max+']';
			}
		},
		/**
		* earfch转化
		* @param e{event}: 鼠标事件
		* @param props{object}: 属性集合
		**/
		earfchBlur: function(e,props){
			var titleDom = $('#'+props.name).parent().prev();
			titleDom.text(titleDom.text().replace(/\(\w+\)/g,''));

			var val = $('#'+props.name).numberbox('getValue'),
				oldVal = $('#'+props.name).attr('oldvalue'),
				sufVal = val?translateToFre(val):'',
				suffstr = sufVal?sufVal.substring(sufVal.indexOf('(')):'';
			
			$('#'+props.name).numberbox({suffix: suffstr.replace(')',')')});
			
			if(props.cascade) {
				try{
					var cascade = eval('('+props.cascade+')'),
						item = cascade[0];
					
					var relys = item.rely.split(','),
						bandVal = $('#'+relys[0]).textbox('getValue'),
						result = checkEarfcnByBand(val, bandVal);
					
					if(result.valid || oldVal == val) {
						return '';
					}else {
						return '<%=rb.getString("PinLv")%> <%=rb.getString("FanWei")%>: ' + JSON.stringify(result.range);
					}
				}catch(e){}
			}
		},
		/**
		* earfch聚焦事件
		* @param e{event}: 鼠标事件
		**/
		earfchFocus: function(e){
			var tb = $(e.target).parent().prev();
			var val = tb.numberbox('getValue');
			tb.numberbox('setText',val);
		},
		/**
		* plmn校验
		* @param e{event}: 鼠标事件
		* @param props{object}: 属性集合
		**/
		plmnBlur: function(e,props){
			var val = $('#'+props.name).textbox('getValue')||'',
				inValid = false,
				regInt = /^-?\d+$/;
			
			if(val && !regInt.test(val)) inValid = true;
			if(val){
				var pureVal = val.replace('-','');
				if(pureVal.length>6) inValid = true;
				if(pureVal.length<5) inValid = true;
			}
			
			if(inValid) return '<%=rb.getString("PLMNFanWeiTiShi")%>';
		},
		/**
		* imsi ip变动更新
		* @param newVal{string}: 当前值
		* @param oldVal{string}: 历史值
		* @param props{object}: 属性集合
		**/
		imsiIpChange: function(newVal,oldVal,props){
			var ctn = $('#'+props.name).parents('.form-item');
			$('.flex-ctn-row',ctn).remove();

			if(newVal){
				newVal.split(',').map(function(ipVal){
					eNbSetting.addImsiIp(ipVal,ctn);
				});
			}
		},
		/**
		* imsi ip校验
		* @param e{event}: 鼠标事件
		* @param props{object}: 属性集合
		**/
		imsiIpBlur: function (e,props){
			var leftVal = $('#'+props.name+'_left').textbox('getValue'),
				rightVal = $('#'+props.name).textbox('getValue'),
				nowVal = $('#'+props.name.replace('_show','')).textbox('getValue');
			// 校验IMSI合法性
			if(leftVal && leftVal.length!=15 || isNaN(leftVal)) return '<%=rb.getString("LGWImsiChangDuCuoWu")%>';
			// 校验ip合法性
			if(rightVal && !isValidIP(rightVal)){
				return '<%=rb.getString("IPDiZhiFeiFa")%>';
			}
			// imsi ip输入合法时
			if(leftVal && rightVal && leftVal.length==15 && isValidIP(rightVal)){
				var list = nowVal.split(','),
					combVal = leftVal +'+'+ rightVal
					existed = false;
				list.map(function(ipItem){
					if(ipItem.indexOf(combVal.trim())>=0) existed = true;
				});
				if(existed && props.button) return '<%=rb.getString("YiCunZai")%>';
			}
		},
		/**
		* ntpServer校验
		* @param e{event}: 鼠标事件
		* @param props{object}: 属性集合
		**/
		ntpServerBlur: function(e,props){
			var val = $('#'+props.name).textbox('getValue'),
				reg = /^[a-zA-Z0-9-]+(\.[a-zA-Z0-9-]+)+$/;
			if(reg.test(val)){

			}else if(val){
				return '<%=rb.getString("ntpServerGeShi")%>'
			}
		},
		/**
		* spset校验
		* @param e{event}: 鼠标事件
		* @param props{object}: 属性集合
		**/
		ipsetBlur: function(e,props){
			var val = $('#'+props.name).textbox('getValue');

			if(val && !isValidIP(val)){
				return '<%=rb.getString("IPDiZhiFeiFa")%>';
			}
		},
		/**
		* ip范围左域校验
		* @param e{event}: 鼠标事件
		* @param props{object}: 属性集合
		**/
		leftIpBlur: function(e,props){
			var options = $('#'+props.name).textbox('options'),
				ipSelf = $('#'+props.name).textbox('getValue'),
				ipRight = '',
				origVal = options.originalValue;

			var msg = eNbSetting.staticIpBlur(e,props,ipSelf);
			if(msg) return msg;
			/* 与右ip比较 */
			if(props.cascade) {
				var cascade = eval('('+props.cascade+')');
				cascade.map(function(item){
					var relys = item.rely.split(',');
					ipRight = $('#'+relys[2]).textbox('getValue');
				})
			}
			if(eNbSetting.compareIp(ipSelf,ipRight)) return '<%=rb.getString("IPYingXiaoYu")%>';
		},
		/**
		* ip范围右域校验
		* @param e{event}: 鼠标事件
		* @param props{object}: 属性集合
		**/
		rightIpBlur: function(e,props){
			var options = $('#'+props.name).textbox('options'),
			ipSelf = $('#'+props.name).textbox('getValue'),
			ipLeft = '',
			origVal = options.originalValue;

			var msg = eNbSetting.staticIpBlur(e,props,ipSelf);
			if(msg) return msg;
			/* 与左ip比较 */
			if(props.cascade) {
				var cascade = eval('('+props.cascade+')');
				cascade.map(function(item){
				var relys = item.rely.split(',');
				ipLeft = $('#'+relys[2]).textbox('getValue');
				})
			}
			if(eNbSetting.compareIp(ipLeft,ipSelf)) return '<%=rb.getString("IPYingDaYu")%>';
		},
		/**
		* 静态ip校验
		* @param e{event}: 鼠标事件
		* @param props{object}: 属性集合
		* @param value{string}: 当前值
		**/
		staticIpBlur: function(e,props,value){/* 左右ip的依赖的 ip和子网掩码校验 */
			var invalid = false,
				msg = '';
			if(props.cascade) {
				var cascade = eval('('+props.cascade+')');
				cascade.map(function(item){
					var relys = item.rely.split(',');
					var ip = $('#'+relys[0]).textbox('getValue'),
						mask = $('#'+relys[1]).combobox('getValue'),
						minIp = getLowAddr(ip,mask),
						maxIp = getHighAddr(ip,mask);
					// 合法校验
					if(ip && mask){
						if(isValidIP(ip)){
							if(eNbSetting.compareIp(value,minIp) && eNbSetting.compareIp(maxIp,value)){

							}else{
								invalid = true;
								msg = '<%=rb.getString("IPFanWei")%> '+minIp+'-'+maxIp;
							}
						}else{
							invalid = true;
							msg = '<%=rb.getString("IPDiZhiFeiFa")%>';
						}
					}else{
						invalid = true;
						msg = '<%=rb.getString("IPMaskBiTian")%>';
					}
				})
			}
			if(invalid) return msg;
		},
		/**
		* ip大小比对
		* @param ipA{string}: ip值
		* @param ipB{string}: ip值
		**/
		compareIp: function(ipA, ipB){
			ipA = ipA.split('.').map(function(item){
				return padLeft(item,3,'0');
			});
			ipB = ipB.split('.').map(function(item){
				return padLeft(item,3,'0');
			});
			return ipA.join('')>=ipB.join('');

			function padLeft (str, len, charStr) {
				var s = str + '';
				return new Array(len - s.length + 1).join(charStr,  '') + s;
			}
		},
		/**
		* imsi ip添加逻辑及校验
		* @param props{object}: 属性集合
		**/
		imsiIpClick: function (props){
			var ctn = $('#'+props.name).parents('.form-item'),
				hideVal = $('#'+props.name).textbox('getValue'),
				leftVal = $.trim($('#'+props.name+'_show_left').textbox('getValue')),
				rightVal = $.trim($('#'+props.name+'_show').textbox('getValue'));

			if(leftVal && !isNaN(leftVal) && leftVal.length==15 && rightVal && isValidIP(rightVal)) {
				var isValid = true;
				if(props.cascade) {
					// 依赖项校验
					var cascade = eval('('+props.cascade+')');
					cascade.map(function(item){
						var relys = item.rely.split(','),
							leftIp = $.trim($('#'+relys[0]).textbox('getValue')),
							rightIp = $.trim($('#'+relys[1]).textbox('getValue'));
						if(isValidIP(leftIp) && isValidIP(rightIp) && eNbSetting.compareIp(rightVal,leftIp) && eNbSetting.compareIp(rightIp,rightVal)){
						}else {
							isValid = false;
							ctn.attr('data-msg','<%=rb.getString("IPFanWei")%> '+leftIp+'-'+rightIp).addClass('invalid');
							setTimeout(function(){
								$('#'+props.name).next().find('input').blur();
							},3000);
						}
					})
				}
				if(isValid){
					var combVal = leftVal +'+'+rightVal;
					if(hideVal){
						var list = hideVal.split(','),
							existed = false;
						list.map(function(ipItem){
							if(ipItem.indexOf(combVal.trim())>=0) existed = true;
						});
						if(existed) return;
						else hideVal += ','+combVal;
					}else{
						hideVal = combVal;
					}

					eNbSetting.addImsiIp(combVal,ctn);
					$('#'+props.name).textbox('setValue',hideVal);
					$('#'+props.name+'_show_left').textbox('setValue','');
					$('#'+props.name+'_show').textbox('setValue','');
				}
			}
			if(!leftVal || !rightVal){
				ctn.attr('data-msg','<%=rb.getString("IPAndIMSITiShi")%>').addClass('invalid');
				setTimeout(function(){
					$('#'+props.name).next().find('input').blur();
				},3000);
			}
		},
		/**
		* imsi ip添加
		* @param val{string}: 当前值
		* @param ctn{dom}: dom容器节点
		**/
		addImsiIp: function (val,ctn){
			var suffCtn = $('.flex-ctn-row',ctn);
			if(suffCtn.length==0){
				suffCtn = $('<div class="flex-ctn-row" style="padding-left: 3px;"></div>');
				ctn.append(suffCtn);
			}

			var suffix = $('<div class="form-imsi-suffix"><span class="text">'+val+'</span><span class="form-bt-remove el-icon el-icon-operation-delete"></span></div>');
			// 删除按钮事件绑定
			suffix.find('.form-bt-remove').on('click',function(){
				suffix.remove();
				var tb = ctn.find('input[textboxname]:first'), ipval = '';

				$('.flex-ctn-row>.form-imsi-suffix',ctn).each(function(n,item){
					if(ipval) ipval += ',';
					ipval += $(item).find('.text').text();
				});
				tb.textbox('setValue',ipval);
				tb.next().find('input').blur();
			});

			suffCtn.append(suffix);
		},
		/**
		* ip重置
		* @param newVal{string}: 当前值
		* @param oldVal{string}: 历史值
		* @param props{object}: 属性集合
		**/
		resetIp: function(newVal,oldVal,props){
			if(props.cascade) {
				var cascade = eval('('+props.cascade+')');
				cascade.map(function(item){
					/* 重置级联项并触发校验 */
					if(item.target){
						item.target.split(',').map(function(tarId){
							var ctn = $('#'+tarId),
							pros = ctn.data('props'),
							fnName = $.domRenderDefaults[pros.type],
							orvalue = ctn[fnName]('options').originalValue;
							
							if(pros.type == 'range') {
								var cas = eval('('+pros.cascade+')'),
									relys = cas[0].rely.split(','),
									leftIP = $('#'+relys[0]).textbox('getValue'),
									rightIP = $('#'+relys[1]).textbox('getValue');
								// 判断依赖ip范围是否合法，并过滤不在此范围的IMSI+IP
								if(isValidIP(leftIP) && isValidIP(rightIP)) {
									var rangeValue = ctn.textbox('getValue');
									if(rangeValue) {
										var result = rangeValue.split(',').filter(function(item){
												var imsiIP = item.split('+')[1];
												return eNbSetting.compareIp(imsiIP,leftIP) && eNbSetting.compareIp(rightIP,imsiIP);
											});
										ctn[fnName]('setValue',result.join(','));
									}
								}else {
									ctn[fnName]('setValue',orvalue);
								}
							}

							//ctn[fnName]('setValue',orvalue);
							ctn.next().find('input').blur();
						});
					}
					/* 触发依赖项校验 */
					if(item.rely){
						item.rely.split(',').map(function(relyId){
							$('#'+relyId).next().find('input').blur();
						});
					}
				})
			}
		},
		/**
		* imsi ip重置
		* @param ctn{dom}: dom容器节点
		**/
		resetIMSIIP: function(ctn){
			var orvalue = $(ctn).textbox('options').originalValue;
			$(ctn).textbox('setValue',orvalue);
		},
		/**
		* LBT Time校验
		* @param e{event}: 鼠标事件
		* @param props{object}: 属性集合
		**/
		LBTTimeBlur:function(e,props){
			var reg = /^((20|21|22|23|[0-1]\d):[0-5]\d:[0-5]\d)$/;
			var value = $('#'+props.name).textbox('getValue');
			var nowVal = $('#'+props.name.replace('_show','')).textbox('getValue')
			if(value == ""){
				
			}else{
				if(reg.test(value)){
					
				}else{
					return "<%=rb.getString("LBTChuFaShiJianFanWei")%>";
				} 
			}
		},
		/**
		* 子网掩码校验
		* @param e{event}: 鼠标事件
		* @param props{object}: 属性集合
		**/
		maskValid: function(e,props){
			var regstr = /^((128|192)|2(24|4[08]|5[245]))(\.(0|(128|192)|2((24)|(4[08])|(5[245])))){3}$/,
				reg = new RegExp(regstr),
				value = $('#'+props.name).textbox('getValue'),
				originVal = $('#'+props.name.replace('_show','')).textbox('options').originalValue;
			
			var ctner = $('#'+props.name).parents('.form-item');
			if(reg.test(value)){// ip 是否重复
				if(props.button) $('#'+props.name.replace('_show','')).next().find('input').blur();
				return '';
			}else if(value != '' && value!=originVal){
				return '<%=rb.getString("QingShuRuHeFaDeYanMa")%>';
			}else{
				if(props.button) $('#'+props.name.replace('_show','')).next().find('input').blur();
			}
		},
		/**
		* TFT List 端口校验
		* @param e{event}: 鼠标事件
		* @param props{object}: 属性集合
		**/
		TFTListPortBlur:function(e,props){
			var portVal = $('#'+props.name).textbox('getValue');
			
			if(props.cascade) {
				var cascade = eval('('+props.cascade+')');
				cascade.map(function(item){
					var relys = item.rely.split(',');
					var protocolVal = $('#'+relys[0]).combobox('getValue');
					if(protocolVal == 'TCP' || protocolVal == 'UDP'){
						if(portVal){
							if(isNumeric(portVal)&& parseInt(portVal)>=0 && parseInt(portVal)<=65535){
								return ''
							}else{
								return '<%=rb.getString("LGWPortTiShi")%>'
							}
						}else{
							return '<%=rb.getString("BiTian")%>'
						}
					}else{
						return ''
					}
						
				})
			}
		},
		/**
		* IP/Mask 校验
		* @param e{event}: 鼠标事件
		* @param props{object}: 属性集合
		**/
		IPMaskBlur:function(e,props){
			var IPMaskVal = $('#'+props.name).textbox('getValue'),
				regstr = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\/([0-9]|[12][0-9]|3[012])$/,
				reg = new RegExp(regstr);
			if(IPMaskVal == '' || IPMaskVal == null){
				return '<%=rb.getString("BiTian")%>'
			}else if(reg.test(IPMaskVal)){
				
			}else{
				return '<%=rb.getString("IPMaskGeShiTiShi")%>'
			}
		},
		/**
		* 读取配置模板点击事件
		* @param data{object}: 全局数据
		* @param item{object}: 按钮本身属性
		**/
		readTemplateClick:function(data,item){
			var form = $('<form id="readTemplateForm" class="flex-ctn"  style="height: 100%;"></form>');
			$('.form-ctn .form-props').html('').append(form);
			// 加载模板信息的jsp片段
			form.load($.renderDefaultOptions.system+item.url,function(data){
				$.parser.parse(this);
				openPropsPanel();
				eNbSetting.changeTitle(item.title);
				eventBus.$emit('read-init');
			});
		},
		/**
		* 保存模板点击事件
		* @param data{object}: 全局数据
		* @param item{object}: 按钮本身属性
		**/
		saveTemplateClick:function(data,item){
			var tftTb = $('#17EF1A3DD71826B1E0DCED9FDDA82005'),
				qosTb = $('#C5F614F36FAFBAA281777B754312924D'),
				tftRows=[],qosRows=[],tftList=[],qosList=[],
				urls = item.url;
			tftRows = tftTb.datagrid('getRows');
			qosRows = qosTb.datagrid('getRows');
			tftRows.map((item)=>{
				var itemData={};
				itemData.pfId = item['BC3DC315DF2D8654F8995B0D8696C800'];
				itemData.index = item['BC3DC315DF2D8654F8995B0D8696C800'];
				itemData.appName = item['A0A25A7E46F515D47C97BA3D733F32B3'];
				itemData.protocal = item["CB674AAEFC770A68A7D3D5FEC007238E"];
				itemData.ipMask = item["827AE949AC32A97C1FA6F73AEA35A6C8"];
				itemData.port = item["F1A1CB61FAA12C13DF9E1E24B2B4EAF0"];
				tftList.push(itemData);
			})
			qosRows.map((item)=>{
				var itemData={};
				itemData.pccId = item["D303A3555BED53E8FC5F34E883104904"];
				itemData.pccName = item["56C85E6D9E1F0229854130A5FAA47FFC"];
				itemData.index = item["B4658365E4D73CD935B128E64F14F6AB"];
				itemData.qci = item["BE5E24E4B68D8037D68B7047C13C0A93"];
				itemData.mbrUl = item["702A697631DA87471CFBF0F6AE1B2DDA"];
				itemData.mbrDl = item["A175EC59C27D988CED837D8A31D1AFD6"];
				itemData.grbUl = item["D4BD7B7FBCE7FE09C76FF9B2B0B63DF9"];
				itemData.grbDl = item["0EAB6B40D32DBF65DE0A93345B517E01"];
				itemData.arpPl = item["BB88A1C448BE32514BD74C8F6296B8F1"];
				itemData.arpPci = item["FA7036C847AD76E239E9A0538CA0EE37"];
				itemData.arpPvi = item["D4C8710204B6D3D3EA5DF0D34C36B3E6"];
				itemData.precedence = item["B4658365E4D73CD935B128E64F14F6AB"];
				itemData.pfList = item["5A5BA201950EDCAB6F15E29DA226185A"];
				qosList.push(itemData);
			})
			var params={
				tftList:tftList,
				qosList:qosList
			};
			params = JSON.stringify(params);
			eventBus.$confirm('<%=rb.getString("QoSMuBanBaoCunTiShi")%>','<%=rb.getString("QueRen")%>').then(function(){
				axios.post(urls,params,{headers:{'Content-Type':'application/json;charset=utf-8'},}).then(function(response){
					var data = response.data;
					if(data) {
						if(data["success"]){
							eventBus.$message({
								message: '<%=rb.getString("ChengGong")%>',
								type:'success'
							});
							
						}else{
							eventBus.$message.error(data["message"])
						}
					}
				}).catch(function(error){})
			});
		},
		/**
		* TFT 添加事件
		* @param opts{object}: 属性集合
		**/
		TFTAddClick(opts){
			var tb = $('#17EF1A3DD71826B1E0DCED9FDDA82005');

			var rows = tb.datagrid('getRows');
			if(rows.length>=32) {
				return ;
			}
			var form = $('<form id="TFTAddForm" class="flex-ctn" operateType="add" style="height: 100%;"></form>');
			$('.form-ctn .form-props').html('').append(form);
			// 加载TFT新增页面的jsp片段
			form.load($.renderDefaultOptions.system+opts.propsUrl,function(data){
				$.parser.parse(this);
				openPropsPanel();
				eNbSetting.changeTitle(opts.label);
				eventBus.$emit('tft-init',opts,'','add');
			});
		},
		/**
		* TFT 修改事件
		* @param opts{object}: 属性集合
		* @param row{object}: 表格行数据
		**/
		TFTEditClick(opts,row){
			var form = $('<form id="TFTAddForm" class="flex-ctn" operateType="edit" style="height: 100%;"></form>');
			$('.form-ctn .form-props').html('').append(form);
			// 加载TFT新增页面的jsp片段
			form.load($.renderDefaultOptions.system+opts.propsUrl,function(data){
				$.parser.parse(this);
				openPropsPanel();
				eNbSetting.changeTitle(opts.label);
				eventBus.$emit('tft-init',opts,row,'edit');
			});
		},
		/**
		* QoS 添加事件
		* @param opts{object}: 属性集合
		**/
		QoSAddClick(opts){
			var tftTb = $('#17EF1A3DD71826B1E0DCED9FDDA82005'),
				qosTb = $('#C5F614F36FAFBAA281777B754312924D'),
				tftRows=[],qosRows=[];
			tftRows = tftTb.datagrid('getRows');
			qosRows = qosTb.datagrid('getRows');
			if(qosRows.length>=32) {
				var title = '<%=rb.getString("ZuiDuoCunZai")%>'+'<%=rb.getString("TiaoShu")%>' +': '+ 32 ;
				eventBus.$message.error(title)
				return ;
			}
			if(tftRows.length == 0) {
				var title = '<%=rb.getString("QingChuangJianTFT")%>';
				eventBus.$message.error(title)
				return ;
			}
			var form = $('<form id="QoSAddForm" class="flex-ctn" operateType="add" style="height: 100%;"></form>');
			$('.form-ctn .form-props').html('').append(form);
			// 加载QoS新增页面的jsp片段
			form.load($.renderDefaultOptions.system+opts.propsUrl,function(data){
				$.parser.parse(this);
				openPropsPanel();
				eNbSetting.changeTitle(opts.label);
				eventBus.$emit('qos-init',opts,'','add');
			});
		},
		/**
		* QoS 修改事件
		* @param opts{object}: 属性集合
		* @param row{object}: 表格行数据
		**/
		QoSEditClick(opts,row){
			var form = $('<form id="QoSAddForm" class="flex-ctn" operateType="edit" style="height: 100%;"></form>');
			$('.form-ctn .form-props').html('').append(form);
			// 加载QoS新增页面的jsp片段
			form.load($.renderDefaultOptions.system+opts.propsUrl,function(data){
				$.parser.parse(this);
				openPropsPanel();
				eNbSetting.changeTitle(opts.label);
				eventBus.$emit('qos-init',opts,row,'edit');
			});
		},
		pMaxValid: function(e,props){
			var val = $('#'+props.name).textbox('getValue'),
				reg = /^(-?)[0-9]+$/;

			if(reg.test(val) && (val == -127 || (val>=-33 && val<=33)) ){
				
			}else if(val){
				return '<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%>-127、[-33,33]<%=rb.getString("JuHao")%><%=rb.getString("ZhengXing")%>'
			}
		},
		rsrqValid: function(e,props){
			var val = $('#'+props.name).textbox('getValue'),
				reg = /^(-?)[0-9]+$/;

			if(reg.test(val) && (val == -1 || (val>=-34 && val<=-3)) ){
				
			}else if(val){
				return '<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%>-1、[-34,-3]<%=rb.getString("JuHao")%><%=rb.getString("ZhengXing")%>'
			}
		},
		mmeBindPlmnClick: function (props){
			var $mmedom = $('#'+props.name+'_show'),
				plnm = $('#'+props.name+'_sufix').val(),
				value = $mmedom.textbox('getValue'),
				ctner = $mmedom.parents('.form-item'),
				comboVal = value.trim()+','+plnm;
			
			// ip校验通过设值
			if(isValidIP(value) && value !='0.0.0.0' && plnm){
				var origVal = $('#'+props.name).textbox('getValue');
				if(origVal){
					var list = origVal.split(';'),
						existed = false;
					
					if(origVal.indexOf(value+',') >= 0) {
						existed = true;
					}

					if(list.length>=16) {
						ctner.attr('data-msg','max: 16').addClass('invalid');
						setTimeout(function(){ $('#'+props.name).next().find('input').blur(); },3000);
						
						return;
					}
					
					if(existed) {
						ctner.attr('data-msg','<%=rb.getString("YiCunZai")%>').addClass('invalid');
						setTimeout(function(){ $('#'+props.name).next().find('input').blur(); },3000);

						return;
					} else {
						origVal += ';'+ comboVal;
					}
				}else{
					origVal = comboVal;
				}
				
				$('#'+props.name).textbox('setValue',origVal);
				$mmedom.textbox('setValue','');
				$('#'+props.name).next().find('input').blur();
			}
			if(value == ''){
				ctner.attr('data-msg','<%=rb.getString("IPShuRuTiShi")%>').addClass('invalid');
				setTimeout(function(){
					$('#'+props.name).next().find('input').blur();
				},3000)
			}
		},
		mmePlmnChange(val,oldVal,props){
			var ctner = $('#'+props.name).parents('.form-item');
			$('.flex-ctn-row',ctner).remove();
			
			if(val){
				var arr = val.split(';');
				arr.map(function(item){
					if(item) eNbSetting.addBindSufix(item, ctner);
				});
			}
		},
		addBindSufix(val, ctner) {
			var sufCtn = $('.flex-ctn-row',ctner);
			if(sufCtn.length==0){
				sufCtn = $('<div class="flex-ctn-row" style="padding-left: 3px;"></div>');
				ctner.append(sufCtn);
			}

			var arr = val.split(','),
				mmeIp = arr[0],
				plmn = arr[1],
				itemStr = [
					'<div class="mmeip-plmn-item">',
						'<span class="mmeip">' + mmeIp + '</span>',
						'<span class="plmn">PLMN: ' + plmn + '</span>',
						'<span class="el-icon el-icon-close" style="font-size: 12px;" onclick="removeBind(this, &quot;'+mmeIp+'&quot;, '+plmn+')"></span>',
					'</div>'
				].join(' '),
				suffix = $(itemStr);

			// 删除绑定事件
			suffix.find('.el-icon-close').on('click',function(){
				suffix.remove();
				var tb = ctner.find('input[textboxname]:first'), 
					mmeipPlmn = tb.textbox('getValue'),
					mplist = mmeipPlmn.split(';');
				
				mplist.remove(val);

				tb.textbox('setValue',mplist.join(';'));
				tb.next().find('input').blur();
			});

			sufCtn.append(suffix);
		},
		plmnClick: function(props){
			var $plmn = $('#'+props.name+'_show')
				value = $plmn.textbox('getValue'),
				ctner = $plmn.parents('.form-item'),
				reg = /^\d{5,6}$/;

			// ip校验通过设值
			if(reg.test(value) && !['00000','000000'].includes(value)){
				var origVal = $('#'+props.name).textbox('getValue');
				if(origVal){
					var list = origVal.split(','),
						existed = false;
					
					if(list.includes(value.trim())) {
						existed = true;
					}

					if(list.length>=6) {
						ctner.attr('data-msg','max: 6').addClass('invalid');
						setTimeout(function(){
							$('#'+props.name).next().find('input').blur();
						},3000);
						
						return;
					}
					
					if(existed) return;
					else origVal += ','+value;
				}else{
					origVal = value;
				}

				$('#'+props.name).textbox('setValue',origVal);
				$('#'+props.name+'_show').textbox('setValue','');
				$('#'+props.name).next().find('input').blur();
			} else {
				ctner.attr('data-msg','Number and length: 5-6').addClass('invalid');
				setTimeout(function(){
					$('#'+props.name).next().find('input').blur();
				},3000);
			}
		},
		plmnChange(val,oldVal,props){
			var ctner = $('#'+props.name).parents('.form-item');
			$('.flex-ctn-row',ctner).remove();
			
			if(val){
				var arr = val.split(',');
				arr.map(function(item){
					if(item) eNbSetting.addPlmnSuffix(item, ctner, props);
				});
			}
		},
		addPlmnSuffix: function(val, ctner, props){
			var sufCtn = $('.flex-ctn-row',ctner);
			if(sufCtn.length==0){
				sufCtn = $('<div class="flex-ctn-row" style="padding-left: 3px;"></div>');
				ctner.append(sufCtn);
			}

			var suffix = $('<div class="form-suffix"><span class="text">'+val+'</span> <span class="form-bt-remove el-icon el-icon-operation-delete"></span></div>');
			// 删除绑定事件
			suffix.find('.form-bt-remove').on('click',function(){
				if(eNbSetting.isPlmnDeletable(val, props)) {
					suffix.remove();
					var tb = ctner.find('input[textboxname]:first'), ipval = '';
					
					$('.flex-ctn-row>.form-suffix',ctner).each(function(n,item){
						if(ipval) ipval += ',';
						ipval += $(item).find('.text').text();
					});
					tb.textbox('setValue',ipval);
					tb.next().find('input').blur();

					eNbSetting.removeBindSelect(val, props);
				}else {
					ctner.attr('data-msg','Binded').addClass('invalid');
					setTimeout(function(){
						$('#'+props.name).next().find('input').blur();
					},3000);
				}
			});

			sufCtn.append(suffix);
			eNbSetting.addBindSelect(val, props);
		},
		addBindSelect(val, props) {
			var cas = props.cascade;
			if(cas) {
				cas = eval('('+cas+')');

				var target = cas[0].target,
					$select = $('#'+target+'_sufix')
					options = $select.children(),
					list = Array.from(options).map(function(item){
						return item.innerText;
					});

				if(!list.includes(val)) $select.append('<option>'+val+'</option>');
			}
		},
		isPlmnDeletable(val, props) {
			var cas = props.cascade,
				bool = true;
			if(cas) {
				try{
					cas = eval('('+cas+')');
	
					var target = cas[0].target,
						$mmeplmn = $('#'+target)
						list = $mmeplmn.textbox('getValue').split(';').map(function(item){
							var arr = item.split(',');
							return arr[1];
						});
	
					if(list.includes(val)) {
						bool = false;
					}
				}catch(e){}
			}

			return bool;
		},
		removeBindSelect(val, props) {
			var cas = props.cascade;
			if(cas) {
				cas = eval('('+cas+')');

				var target = cas[0].target,
					$select = $('#'+target+'_sufix'),
					options = $select.children();

				Array.from(options).map(function(item){
					if(item.innerText == val) item.remove();
				});
			}
		},
		epcChange(newVal,oldVal,props) {
			var vm = this,
				halobEnable = halobSwitchFlag == '1',
				epcEnable = newVal == '1';

			if(props.cascade[0]) {
				var cas = eval('('+props.cascade+')'),
					list = cas[0].target.split(',');

				list.map(function(id){
					var formItem = $('#'+id).parents('.form-item:first');

					if(epcEnable) {
						formItem.addClass('disabled');
					}else {
						formItem.removeClass('disabled');
					}
				});
			}
		},
		autoECIChange(newVal,oldVal,props) {
			var vm = this,
				epcEnable = newVal == '1';

			if(props.cascade[0]) {
				var cas = eval('('+props.cascade+')'),
					list = cas[0].target.split(',');

				list.map(function(id){
					var $dom = $('#'+id);
					
					if($dom.length) {
						var formItem = $dom.parents('.form-item:first');

						if(epcEnable) {
							formItem.addClass('disabled');
							formItem.find('span').addClass('textbox-readonly');
							formItem.find('input').attr('readonly',true);
						}else {
							formItem.removeClass('disabled');
							formItem.find('span').removeClass('textbox-readonly');
							formItem.find('input').attr('readonly',false);
						}
					}
				});
			}
		},
		prachFreqBlur: function(e,props){
			var val = $('#'+props.name).textbox('getValue')||'',
				inValid = false,
				regInt = /^-?\d+$/,
				row = settingVue.selectedRow,
				bandwidth = row?row.bandwidth:'';
			
			if(val) {
				if(bandwidth == '5MHz') {
					if(!regInt.test(val) || val - 25 > 0) inValid = true;
				}
				if(bandwidth == '10MHz') {
					if(!regInt.test(val) || val - 4 < 0 || val - 40 > 0) inValid = true;
				}
				if(bandwidth == '15MHz') {
					if(!regInt.test(val) || val - 75 > 0) inValid = true;
				}
				if(bandwidth == '20MHz') {
					if(!regInt.test(val) || val - 5 < 0 || val - 89 > 0) inValid = true;
				}
			}
			
			if(inValid) return '<%=rb.getString("FanWei")%>: 10MHz: 4-40  20MHz: 5-89  5MHz: <25  15MHz: <75';
		},
		tftBlur(row, tb, props) {
			var pfId = row['7A8D21C159139F517BB9B592CF8E9ACA']+'',
				qosTb = $('#C5F614F36FAFBAA281777B754312924D'),
				has = false;

			if(qosTb && qosTb.length) {
				var rows = qosTb.datagrid('getRows');
				
				rows.map(function(item){
					if(item['5A5BA201950EDCAB6F15E29DA226185A']) {
						var list = item['5A5BA201950EDCAB6F15E29DA226185A'].split(',');

						if(list.includes(pfId)) has = true;
					}
				});
			}

			if(has) {
				showMsg('error_msg', 'PF_ID is used,not allowed to delete!');
				return false;
			}else {
				return true;
			}
		}
	};

    function validMult(str){
		var valid = true, min = 0, max = 503,
			reg = /^(\d+)\.\.(\d+)$/, intReg = /^-?\d+$/;
		var arr = str.split(',');
		arr.map(function(item){
			item = item.trim();
			if(isNaN(item)){
				if(reg.test(item)){
					var m = item.match(reg), first = m[1], second = m[2];
					if(first-second<0){
						if(first<min || first>max || second<min || second>max) valid = false;
					}else valid = false;
				}else valid = false;
			}else if(item<min || item>max || !intReg.test(item)) valid = false;
		});
		return valid;
	}
	function isNumeric(str) {
			if(str.length==0){
				return false;
			}
			for(var i=0;i<str.length;i++){
				if(str.charAt(i)<"0" || str.charAt(i)>"9"){
					return false;
				}
			}
			return true;  
	}
	function checkEarfcnByBand(earfcn, band) {
		var rels = {
				1: [2110, 2170],      2: [1930, 1990],   3: [1805, 1880],   4: [2110, 2155],      5: [869, 894], 
				6: [875, 885],        7: [2620,2690],    8: [925, 960],     9: [1844.9, 1879.9],  10: [2110, 2170],
				11: [1475.9, 1495.9], 12: [729, 746],    13: [746, 756],    14: [758, 768],       17: [734, 746],
				18: [860, 875],       19: [875, 890],    20: [791, 821],    21: [1495.9, 1510.9], 22: [3510, 3590],
				23: [2180, 2200],     24: [1525, 1559],  25: [1930, 1995],  26: [859, 894],       27: [852, 869],
				28: [758, 803],       29: [717, 728],    30: [2350, 2360],  31: [462.5, 467.5],   32: [1452, 1496],
				33: [1900, 1920],     34: [2010, 2025],  35: [1850, 1910],  36: [1930, 1990],     37: [1880, 1930],
				38: [2570, 2620],     39: [1880, 1920],  40: [2300, 2400],  41: [2496, 2696],     42: [3400, 3600],
				43: [3600, 3800],     44: [703, 803],    45: [1447, 1467],  46: [5150, 5925],     47: [5855, 5925],
				48: [3550, 3700],     49: [3550, 3700],  50: [1432, 1517],  51: [1427, 1432],     52: [3300, 3400],
				53: [2483.5, 2495],   65: [2110, 2200],  66: [2110, 2200],  67: [738, 758],       68: [753, 783],
				69: [2570, 2620],     70: [1995, 2020],  71: [617, 652],    72: [461, 466],       73: [460, 465],
				74: [1475, 1518],     75: [1432, 1517],  76: [1427, 1432],  85: [728, 746],       87: [420, 425],
				88: [422, 427]
			},
			key = (band||'').trim(),
			range = rels[key],
			result = {
				valid: false,
				range: []
			};
		
		if(range) {
			var freqStr = earfcnFormatter(earfcn).replace('MHz','MHZ'),
				freq = freqStr.match(/[0-9.]*MHZ/g)[0].replace('MHZ',''),
				min = range[0], max = range[1];
			
			if(freq >= min && freq <= max) {
				result.valid = true;
			}
			
			result.range = range;
		}else {
			result.valid = true;
		}
		
		return result;
	}
	function setSettingTimeout() {
		resetSettingTimeout();

		settingTimeout = setTimeout(function(){
			$('#setting_main').removeClass('loading')
		},180000);
	}
	function resetSettingTimeout() {
		if(settingTimeout) {
			clearTimeout(settingTimeout);
		}
	}
	// LGW IP池限制
	function isForbiddenIP(ipStr) {
		let trimIPStr = ipStr.replace(/\s/g, ''),
			firstSec = trimIPStr.split('.')[0],
			bool = false;

		if(['0.0.0.0', '255.255.255.255'].includes(trimIPStr)) bool = true;

		if(firstSec == '127' || (firstSec - 224 >= 0 && firstSec - 254 <= 0)) bool = true;

		return bool;
	}	
   
    function RFStatusFormatter(value, rowData, rowIndex){
        var row = rowData || {};

        if (value == null || value == "") {
            return null;
        }
        if (value == '--') {
            return value;
        }
        
        if (value == "on" || value == 1) {
                //CA 模式有两个小区； 射频状态要显示两个图标； 辅小区不能编辑，下发射频开启或关闭，只能从主小区下发； 射频状态两个小区相同； 辅小区跟随主小区
                if(row.platformType == 'Intel_CR_CA' || row.platformType == 'MLN_CA') {// 4860
                    value = "<div class='rfConnItem' onmouseover='toMMEDetail(this,1)' onmouseout='hideMMEDetail()' type='RF'>"+ Kai +"</div>"
                        + "<div class='mmeDetails MMEDetail'>RF Status : <span class='mmeStauts'></span></div>"
                        + "<div class='rfConnItem' onmouseover='toMMEDetail(this,1)' onmouseout='hideMMEDetail()' type='RF' style='margin-left: 10px;'>"+Kai+"</div>"
                        + "<div class='mmeDetails MMEDetail'>RF Status : <span class='mmeStauts'></span></div>";
                }else {
                    value = "<div class='rfConnItem' onmouseover='toMMEDetail(this,1)' onmouseout='hideMMEDetail()' type='RF'>"+Kai+"</div>"
                        + "<div class='mmeDetails MMEDetail'>RF Status : <span class='mmeStauts'></span></div>";
                }
        } else if (value == "off" || value == 0) {
                //CA 模式有两个小区； 射频状态要显示两个图标； 辅小区不能编辑，下发射频开启或关闭，只能从主小区下发； 射频状态两个小区相同； 辅小区跟随主小区
                if(row.platformType == 'Intel_CR_CA' || row.platformType == 'MLN_CA') {// 4860
                    value = "<div class='rfDisconnItem' onmouseover='toMMEDetail(this,0)' onmouseout='hideMMEDetail()' type='RF'>"+Guan+"</div>"
                        + "<div class='mmeDetails MMEDetail'>RF Status : <span class='mmeStauts'></span></div>"
                        + "<div class='rfDisconnItem' onmouseover='toMMEDetail(this,0)' onmouseout='hideMMEDetail()' type='RF' style='margin-left: 10px;'>"+Guan+"</div>"
                        + "<div class='mmeDetails MMEDetail'>RF Status : <span class='mmeStauts'></span></div>";
                }else {
                    value = "<div class='rfDisconnItem' onmouseover='toMMEDetail(this,0)' onmouseout='hideMMEDetail()' type='RF'>"+Guan+"</div>"
                        + "<div class='mmeDetails MMEDetail'>RF Status : <span class='mmeStauts'></span></div>";
                }
                
        } else{
            var rfStatus = value.split(",");
            var rfStatusText ='';
            if(rfStatus[0] == "on" || rfStatus[0] == "1"){
                rfStatusText = rfStatusText + "<div class='rfConnItem' onmouseover='toMMEDetail(this,1)' onmouseout='hideMMEDetail()' type='RF1'>"+Kai+"</div>";
            }
            if(rfStatus[0] == "off" || rfStatus[0] == "0"){
                rfStatusText = rfStatusText + "<div class='rfDisConnItem' onmouseover='toMMEDetail(this,0)' onmouseout='hideMMEDetail()' type='RF1'>"+Guan+"</div>";
            }
            if(rfStatus[1] == "on" || rfStatus[1] == "1"){
                rfStatusText = rfStatusText + "<div class='rfConnItem' style='margin-left:10px;' onmouseover='toMMEDetail(this,1)' onmouseout='hideMMEDetail()' type='RF2'>"+Kai+"</div>";
            }
            if(rfStatus[1] == "off" || rfStatus[1] == "0"){
                rfStatusText = rfStatusText + "<div class='rfDisConnItem' style='margin-left:10px;' onmouseover='toMMEDetail(this,0)' onmouseout='hideMMEDetail()' type='RF2'>"+Guan+"</div>";
            }
                
            if(rfStatus.length == 3) {
                if(rfStatus[2] == "on" || rfStatus[2] == "1"){
                    rfStatusText = rfStatusText + "<div class='rfConnItem' style='margin-left:10px;' onmouseover='toMMEDetail(this,1)' onmouseout='hideMMEDetail()' type='RF3'>"+Kai+"</div>";
                }
                if(rfStatus[2] == "off" || rfStatus[2] == "0"){
                    rfStatusText = rfStatusText + "<div class='rfDisConnItem' style='margin-left:10px;' onmouseover='toMMEDetail(this,0)' onmouseout='hideMMEDetail()' type='RF3'> "+Guan+"</div>";
                }
                
                value = rfStatusText + 
                        "<div class='mmeDetails MME1Detail'>RF1 Status : <span class='mme1Stauts'></span></div>"+ 
                        "<div class='mmeDetails MME2Detail'>RF2 Status : <span class='mme2Stauts'></span></div>"+ 
                        "<div class='mmeDetails MMEDetail'>RF3 Status : <span class='mme3Stauts'></span></div>"; 
            }else {
                value = rfStatusText + "<div class='mmeDetails MME1Detail'>RF1 Status : <span class='mme1Stauts'></span></div>"+ "<div class='mmeDetails MME2Detail'>RF2 Status : <span class='mme2Stauts'></span></div>"; 
            } 
        }
        return value;
    }
    function toMMEDetail(ele,index){
        var thisTop = $(ele).offset().top;
        var allHeight = $(document).height();
        var thisLeft = $(ele).offset().left;
        var allWidth = $(document).width();
        
        if((allHeight - thisTop) < 200){
            $(ele).siblings(".mmeDetails").css("top","-55px");
        }
        if((allWidth - thisLeft) < 200){
            $(ele).siblings(".mmeDetails").css("left","-90px");
        }
        var eleClass = $(ele).attr('type'); 
        if(eleClass.includes("RF")){
            var YiLianJie = '<%= rb.getString("KaiQi")%>';
            var WeiLianJie = '<%= rb.getString("GuanBi")%>';
        }
        if(eleClass.includes("MME")){
            var YiLianJie = '<%= rb.getString("MMEYiLianJie")%>';
            var WeiLianJie = '<%= rb.getString("MMEWeiLianJie")%>';
        }
        $(".mmeDetails").hide();
        if(eleClass == 'RF1' || eleClass == 'MME1'){
            $(ele).siblings(".MME1Detail").fadeToggle();
            if(index == 1){
                $(".mme1Stauts").text(YiLianJie);
            }else if(index == 0){
                $(".mme1Stauts").text(WeiLianJie);
            }
        }else if(eleClass == 'RF2' || eleClass == 'MME2'){
            $(ele).siblings(".MME2Detail").fadeToggle();
            if(index == 1){
                $(".mme2Stauts").text(YiLianJie);
            }else if(index == 0){
                $(".mme2Stauts").text(WeiLianJie);
            }
        }else if(eleClass == 'RF3' || eleClass == 'MME3'){
            $(ele).siblings(".MMEDetail").fadeToggle();
            if(index == 1){
                $(".mme3Stauts").text(YiLianJie);
            }else if(index == 0){
                $(".mme3Stauts").text(WeiLianJie);
            }
        }else if(eleClass == 'RF' || eleClass == 'MME'){
            $(ele).siblings(".MMEDetail").fadeToggle();
            if(index == 1){
                $(".mmeStauts").text(YiLianJie);
            }else if(index == 0){
                $(".mmeStauts").text(WeiLianJie);
            }
        }	
    }

    function hideMMEDetail() {
        $(".mmeDetails").fadeOut(100);
    }
    function getueCpeCountsData(code,eci,pci,earfcn,sn,cellname,divId,is436Q) {
        var url = '${ctx}/cell/ap/toUEDetailPage.action',
            obj ={
                smallCellCode: code,
                eci: eci,
                pci: pci,
                earfcn: earfcn
            };

        enbvm.cpeslide.url = url;
        
        window.sessionStorage.setItem('ueSmallCellCode',code);
        window.sessionStorage.setItem('ueEci',eci);
        window.sessionStorage.setItem('uePci',pci);
        window.sessionStorage.setItem('ueEarfcn',earfcn);
        window.sessionStorage.setItem('snNumber',sn);
        window.sessionStorage.setItem('cellName',cellname);
        window.sessionStorage.setItem('is436Q',is436Q);

        enbvm.$refs.cpeCount.showSlide(obj, function() {
            $("#tabAlarmCli").click();
        });
    }
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
    function compare(propertyName){
        return function(object1,object2){
            var value1=object1[propertyName];
            var value2=object2[propertyName];
            if(value2<value1) return 1;
            else if(value2>value1) return -1;
            else return 0;
        }
    }
    function ipAddrFormatter(value,row,index) {
        if(value) {
            value = '<a href="https://' + value + '" target="_blank" style="color: #4d84ff;">'+value+'</a>';
        }

        return value;
    }
    function cellStateFormatter(value, rowData, rowIndex){
        if (value == null) {
            return null;
        }
        else if (value == "1") {
            val = "<%= rb.getString("JiHuo")%>";
            value = "<div class='activeStatusItem'>"+(val)+"</div>"
        }
        else if (value == "0") {
            //状态不一样展示的文字也不一样
            val = "<%= rb.getString("QuJiHuo")%>";
            value = "<div class='inactiveStatusItem'>"+(val)+"</div>"
        }
        return value;
    }
    /**
    * 判断对象是否为空
    * @param obj{object}: 要判断的对象
    **/
    function isEmptyObject(obj){
        for(var key in obj){
            return false;
        }
        return true;
    }
    function halobStatusFormatter (value, rowData, rowIndex) {
        if ("1" == value) {
            value = "<span class='el-icon el-icon-status-enable' style='margin-right:10px;font-size: 20px;'></span>"+Kai;
        } else if ("0" == value) {
            value = "<span class='el-icon el-icon-status-disable' style='margin-right:10px;font-size: 20px;'></span>"+Guan;
        } else {
            value = "--";
        }
        return value;
    }
    function kpiStatusFormatter(value,row,index){
        if(value == null){
            return null;
        }else if(value == "off"){
            value = "<span class='el-icon el-icon-status-kpi-off' style='float:left;font-size:20px;margin-right:10px'></span>"+"<span style='color:#444;margin-top:4px;display:block;float:left'>"+(value)+"</span>"		
        }else if(value == "normal"){
            value = "<span class='el-icon el-icon-status-kpi-normal' style='float:left;font-size:20px;margin-right:10px'></span>"+"<span style='color:#444;margin-top:4px;display:block;float:left'>"+(value)+"</span>"
        }else if(value == "broken"){
            value = "<span class='el-icon el-icon-status-kpi-failure' style='float:left;font-size:20px;margin-right:10px'></span>"+"<span style='color:#444;margin-top:4px;display:block;float:left'>"+(value)+"</span>"
        }
        return value;
    }
    function lockStatusFmt(value,rowData,rowIndex){
        if(value == "--"){
            return value;
        }else{
            if(value == 0){
                return "<div><span class='el-icon el-icon-status-unlock' style='margin-right:10px;font-size:20px;'></span><%=rb.getString("YouXiaoQiJieSuo")%></div>";
            }else{
                return "<div><span class='el-icon el-icon-operation-lock yellowIcon' style='margin-right:10px;font-size:20px;'></span><%=rb.getString("YouXiaoQiSuoDing")%></div>";
            }
        }
    }
</script>