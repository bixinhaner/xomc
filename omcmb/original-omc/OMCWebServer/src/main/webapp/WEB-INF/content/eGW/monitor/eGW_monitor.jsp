<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	
	#egwMonitorSlide{
		z-index: 2002!important;
	}
	/* 列表告警显示样式 */
	#egwMonitorPage .alarmListSty{
		display:inline-block;
		min-width:13px;
		padding:0 5px;
		height:23px;
		line-height:24px;
		border-radius:23px;
		text-align:center;
		color:#FFFFFF;
		-webkit-transform:scale(0.8);
		font-size:12px;
		cursor:pointer;
	}
	#egwMonitorPage .alarmCritical{
		background:#E88282;
	}
	#egwMonitorPage .alarmMajor{
		background:#DCAA5E;
	}
	#egwMonitorPage .alarmMinor{
		background:#CCCC66;
	}
	#egwMonitorPage .alarmWarning{
		background:#9AF0FE;
	}
	#egwMonitorPage .loading::before{
		background-color: rgba(255,255,255,1);
	}
	#egwMonitorSlide{
		z-index: 1999!important;
	}
	#egwMonitorSlide .el-card > .el-card__header .clearfix span{
		z-index: 999;
	}
	#egwMonitorPage .abbreviationCls{
		overflow: hidden;
		white-space: nowrap;
		text-overflow: ellipsis;
	}
	#egwMonitorPage .cellNameClass{
		display: inline-block;
		height: 18px;
		width: 18px;
		text-align: center;
		line-height: 18px;
		border: 1px solid #DCDFE6;
		border-radius: 2px;
	}
	#egwMonitorPage .cellNameClass .el-icon::before{
		font-size: 16px;
		color: #F2B354;
	}
	#egwMonitorPage .cellNameClass:hover{
		border: 1px solid #4D84FF;
	}
	#egwMonitorPage .enbModelCls{
		display: inline-block;
		width:18px;
		height:18px;
		background:url(${ctx}/css/images/newIcon/statusIcon/TURBOwu.png) no-repeat;
	}
	#egwMonitorPage .gnbModelCls{
		display: inline-block;
		width:18px;
		height:18px;
		background:url(${ctx}/css/images/newIcon/statusIcon/gNB_model.png) no-repeat;
	}
	#egwMonitorPage .egwModelBoxCls{
		display: flex;
		align-items: center
	}
	#egwMonitorPage .egwModelBoxCls > span{
		margin-right: 5px;
	}
	#egwMonitorPage .el-ctable-toolbar{
		padding: 0px!important;
	}
	.settingSlide {
    	width:70% !important;
    	min-width:800px;
    	left:auto;
    }
	#egwMonitorPage .egwCollectMsgBoxCls {
		display: flex;
		top: 10px;
		right: 60px;
		position: absolute;
		border: 1px solid #DFE2EE;
		border-radius: 5px;
	}
	#egwMonitorPage .egwCollectMsgBoxCls .foldBtnCls{
		height: 26px;
		width: 26px;
		display: flex;
		flex-direction: column;
		justify-content: center;
		align-items: center;
		background-color: #fff;
		cursor: pointer;
		transform: rotate(90deg);
	}
	#egwMonitorPage .egwCollectMsgBoxCls .collect-bt {
		color:#4D84FF;
		margin-left: 15px;
		text-decoration: underline;
		cursor: pointer;
	}
</style>
<div id='egwMonitorPage' style="overflow: auto;">
	<div class="container commonWarp" style="min-width: 900px;">
		<el-ctable
			:url="egwMonitorTableUrl"
			:query-params="queryParams"
			ref="egwMonitorTable"
			id="egwMonitorTable"
			:height="height"
			:time="6"
			:page-size="pageSize"
			:page-list="pageList"
			pagination="true">
				<!-- 列表toolbar -->
			<template slot="toolbar">
				<!-- 导出 -->
				<div class='toolbarHeadBtnBoxCls commonQuery' style='margin-bottom: 0;height:45px;'>
					<div class="newIconBoxCls-bt" style="right:20px;top:12px;" @click="exportEgwDevice" tip="<%=rb.getString("DaoChu")%>">
						<span class="el-icon el-icon-operation-export"></span>
					</div>
					<!-- 收集TR069报文 -->
					<div v-if="isEgwCollectExisted && isAdmin" class="egwCollectMsgBoxCls">
						<div v-if="msgExtend" style="margin-right: 20px;line-height:26px;">
							<span style="padding: 0px 10px;">{{collectActiveSn}}</span>
							<span style="padding: 0px 5px;" v-if="collectTaskTime==''">
								<i class="el-icon el-icon-status-yes" style="font-size: 12px;"></i>
								<%=rb.getString("ChengGong")%>
							</span>
							<span v-if="collectTaskTime!=''" style="display: inline-block;padding: 2px 30px;background: #4d84ff;border-radius: 2px;margin: 0px 5px 2px 5px;"></span>
							<span v-if="collectTaskTime!=''" style="border: 1px solid #e3e3e3;border-radius: 3px;padding: 2px 4px;">
								<span style="cursor: pointer;" @click="stopCollect">
									<i style="padding: 4px;background: red;height: 0px;display: inline-block;border-radius: 3px;"></i>
									<%=rb.getString("TingZhi")%>
								</span>
								<span style="margin-left: 5px;">
									{{collectTaskTime}}
								</span>
							</span>
							<span style="margin-left: 20px;">
								<a class="collect-bt" @click="viewMsg"><%=rb.getString("ChaKan")%></a>
								<a class="collect-bt" @click="downloadMsg"><%=rb.getString("XiaZai")%></a>
								<a class="collect-bt" @click="clearMsg"><%=rb.getString("QingChu")%></a>
							</span>
						</div>
						<div @click="msgExtend = !msgExtend" class="foldBtnCls">
							<span v-if="msgExtend" class="el-icon el-icon-common-query-up"></span>
							<span v-if="!msgExtend" class="el-icon el-icon-common-query-down"></span>
						</div>
					</div>
					<el-query type="normal" @query="queryEgwMonitor" placeholder="<%=rb.getString("eGWBianMa")%> / <%=rb.getString("EGWIP")%>"></el-query>
				</div>
			</template>
				<!-- 列表columns -->
			<el-table-column label="" width="70" class-name="no-text-tips">
				<template slot-scope="scope"><!-- 将元素或组件表示为作用域插槽          -->
					<div class="el-icon el-icon-circle-setting1" @click="openSettingPage(scope.row)" style="font-size:19px;"></div>
                    <div class="el-icon el-icon-operation-more-circle" @click="optClick(scope.row, event)" v-clickoutside="hideMenus"></div>
				</template>
			</el-table-column>
			<el-table-column prop="connectionStatus" width="50" sortable>
				<template slot-scope="scope">
					<div v-html="connStatusFmt(scope.row, scope.row['connectionStatus'], scope.$index)"></div>
				</template>
			</el-table-column>
			<el-table-column prop="alarm_count" label="<%=rb.getString("GaoJingShu")%>" min-width="110" sortable>
				<template slot-scope="scope">
					<div style="text-align: center;">
						<span v-show="scope.row.alarm_serverity == '31001'" @click="goDetail(scope.row)" class='alarmCritical alarmListSty'>{{scope.row.alarm_count}}</span>
						<span v-show="scope.row.alarm_serverity == '31002'" @click="goDetail(scope.row)" class='alarmMajor alarmListSty'>{{scope.row.alarm_count}}</span>
						<span v-show="scope.row.alarm_serverity == '31003'" @click="goDetail(scope.row)" class='alarmMinor alarmListSty'>{{scope.row.alarm_count}}</span>
						<span v-show="scope.row.alarm_serverity == '31004'" @click="goDetail(scope.row)" class='alarmWarning alarmListSty'>{{scope.row.alarm_count}}</span>
						<span v-show="scope.row.alarm_serverity != '31001' && scope.row.alarm_serverity != '31002' && scope.row.alarm_serverity != '31003' && scope.row.alarm_serverity != '31004'">0</span>
					</div>
				</template>
			</el-table-column>
			<el-table-column prop="egwSn" label="<%=rb.getString("eGWBianMa")%>" min-width="150"></el-table-column>
			<el-table-column prop="egwName" label="<%=rb.getString("EGWMingCheng")%>" min-width="150">
				<template slot-scope="scope">
					<div class="abbreviationCls" v-if="scope.row.deviceNameTip === '0'">{{scope.row.egwName}}</div>
					<div v-if="scope.row.deviceNameTip !== '0'">
						<el-popover trigger="click">
							<div slot="reference" class="abbreviationCls">
								<div class="cellNameClass">
									<span class="el-icon el-icon-circle-warning"></span>
								</div>
								{{scope.row.egwName}}
							</div>
							<div style="padding: 15px 20px 10px 20px;">
								<div  style="margin-bottom: 10px;"> <span class="panel_close" @click="closeSyncName" style="margin-right: 0px;"></span></div>
								<div><%=rb.getString("Title_SheBeiMingCheng")%> : {{scope.row.reportHostName}}</div>
								<div ><%=rb.getString("ShiFouTongBuMingChengDaoOMC")%><div>
								<div style="margin-top: 20px;">
									<span class="button_simple" @click="syncName(scope.row.egwCode)">
										<%=rb.getString("QueDing")%>
									</span>
									<span class="button_simple white" @click="closeSyncName"><%=rb.getString("QuXiao")%></span>
								</div>
							</div>
						</el-popover>
					</div>
				</template>
			</el-table-column>
			<el-table-column prop="generation" label="<%=rb.getString("SheBeiXingHao")%>" min-width="120">
				<!--<template slot-scope="scope">
					<div v-if="scope.row.generation == '4G' || !scope.row.generation" class="egwModelBoxCls">
						<span class='enbModelCls' style='font-size:22px'></span>
						<span>4G</span>
					</div>
					<div v-if="scope.row.generation == '5G'" class="egwModelBoxCls">
						<span class='gnbModelCls' style='font-size:22px'></span>
						<span>5G</span>
					</div>
					<div v-if="scope.row.generation == '4G/5G'" class="egwModelBoxCls">
						<span class='enbModelCls' style='font-size:22px'></span>
						<span class='gnbModelCls' style='font-size:22px'></span>
						<span>4G/5G</span>
					</div>
				</template>-->
			</el-table-column>
			<el-table-column prop="egwIp" label="<%=rb.getString("EGWIP")%>" min-width="120"></el-table-column>
			<el-table-column prop="egwPort" label="<%=rb.getString("EGWDuanKou")%>" min-width="110"></el-table-column>
			<el-table-column prop="peerIp" label="<%=rb.getString("HADuiDuanIP")%>" min-width="120"></el-table-column>
			<el-table-column prop="enbNumber" label="<%=rb.getString("WangGuanJiZhanShu")%>(<%=rb.getString("ShiFouZaiXian")%>)" min-width="150"></el-table-column>
			<el-table-column prop="ueCount" label="<%=rb.getString("UEShu")%>(<%=rb.getString("ShiFouZaiXian")%>)" min-width="130"></el-table-column>
			<el-table-column prop="uplink_traffic" label="<%=rb.getString("EGWShangXingLiuLiang")%>" min-width="140"></el-table-column>
			<el-table-column prop="downlink_traffic" label="<%=rb.getString("EGWXiaXingLiuLiang")%>" min-width="160"></el-table-column>
			<el-table-column prop="ha_status" label="<%=rb.getString("ZhuBeiZhuangTai")%>" min-width="150"></el-table-column>
			<el-table-column prop="softwareVersion" label="<%=rb.getString("BanBen")%>" min-width="150"></el-table-column>
			<el-table-column prop="egwDescription" label="<%=rb.getString("MiaoShu")%>" min-width="150"></el-table-column>
		</el-ctable>
		<!-- 菜单 -->
			<el-cmenu ref="menu" @click="menuClick" :data="menus"></el-cmenu>
		<!-- slide -->
		<el-slide ref="egwMonitorSlide" :class="settingslideCls" id="egwMonitorSlide" :url='slideUrl' :title="slideTitle" :footer="slideFooter" :header="slideHeader" :position="slidePosition"
			:height="slideHeight"  :width='slideWidth' @ok="settingSubmit"  @cancel="closeSlide" :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'" >
		</el-slide>
		<el-slide ref="egwSettingLinkSlide" id="egwSettingLinkSlide" :url='settingLinkSlideUrl' :title="settingLinkSlideTitle" :footer="settingLinkSlideFooter" :header="settingLinkSlideHeader" 
			:position="settingLinkSlidePosition" :height="settingLinkSlideHeight"  :width='settingLinkSlideWidth' @ok="settingLinkSubmit"  @cancel="closeSettingLinkSlide" :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'" >
		</el-slide>
		<el-slide ref="egwSettingPage" class="settingSlide" 
           	:url="settingUrl" width="80%"
            :footer="false" 
            :header="false">
        </el-slide>
		<!-- 收集TR069消息报文 -->
		<el-dialog title="<%=rb.getString("QueRen")%>" top="30vh" width="550"
			:visible.sync="collectMessageShow" 
			:modal="false"
			:close-on-click-modal="false">
			<el-form :model="collectForm">
				<div>{{confirmTips}}</div>
				<el-form-item label='<%=rb.getString("ChiXuShiChang")%>' style="display: flex;align-items: center;margin: 5px 0px;">
					<el-select v-model="collectForm.collectInterval" placeholder="Select time" class="collect-select">
						<el-option label="05" value="05"></el-option>
						<el-option label="10" value="10"></el-option>
					</el-select>
					<div style="display: inline;padding: 5px;margin-left: -4px;border: 1px solid #e9e9e9;background: #F5F7FA;"><%=rb.getString("ANRFenZhong")%></div>
				</el-form-item>
				<span v-if="collectExisted">
					<span style="color: #B3B3B3;"><%=rb.getString("ShouJiBaoWenFuGaiTiShi")%> SN={{existedMsgSN}}. </span>
				</span>
			</el-form>

			<div slot="footer" style="text-align: right;">
				<el-button type="primary" @click="sendCollect"><%=rb.getString("QueDing")%></el-button>
				<el-button @click="collectMessageShow = false"><%=rb.getString("QuXiao")%></el-button>
			</div>
		</el-dialog>
		<!-- 收集TR069消息报文详情 -->
		<el-dialog :visible.sync="collectInfoShow" top="10vh">
			<div slot="title">
				<span class="el-dialog__title"><%=rb.getString("XinXi")%></span>
				<i class="el-icon el-icon-operation-export" style="position: absolute;right: 45px;top: 9px;" @click="downloadMsg"></i>
			</div>
			<el-input v-model="collectContent" type="textarea" rows="25" readonly class="none-border"></el-input>
		</el-dialog>
    </div>
</div>
<script type="text/javascript">
var egwStatusTimer;
var egwCollectInterval = null;
var egwMonitor = new Vue({
	el:'#egwMonitorPage',
	data(){
		return {
            queryParams:{
                timeZone:timeZone,
                search_text:'',
            },
			slideUrl:'',
			slideTitle:'',
			slideHeader:'',
			slideFooter:'',
			slidePosition:'',
			slideHeight:'',
			slideWidth:'',

			settingLinkSlideUrl:'',
			settingLinkSlideTitle:'',
			settingLinkSlideHeader:'',
			settingLinkSlideFooter:'',
			settingLinkSlidePosition:'',
			settingLinkSlideHeight:'',
			settingLinkSlideWidth:'',

            height:'100%',
            pageSize:50,
			pageList:[50,100,200],
            menus:[],
            rowData:'',
            egwMonitorTableUrl:'${ctx}/egw/monitor/getEgwMonitorPageList.action',
			settingslideCls:'',
			settingUrl:'',

			collectExisted: false,
			existedMsgSN: '',
			collectMessageShow: false,
			collectForm: {
				collectInterval: ''
			},
			isEgwCollectExisted: false,
			collectDeviceCode: '',
			collectActiveSn: '',

			collectTaskTime: '',
			msgExtend: false,
			collectInfoShow: false,
			collectContent: '',
		}
	},
    computed:{
		operationShow() {
			return writableMap['CODE_EGW'] == true;
		},
		isAdmin() {
			return is_super_user == 'true'
		},
		confirmTips() {
			var vm = this,
				sn = vm.rowData.serial_number,
				msg = '<%=rb.getString("QueRenShouJiPre")%>';

			return msg.replace('placeholder', sn);
		},
	},
	methods:{
		// 表格操作项 打开操作菜单
		optClick(row,evt){
			var vm = this,
				tongbuDisableFlag = true,
				chongQidisableFlag = true,
				rizhidisableflag = true;

			vm.rowData = row;
			if(row.syncResult != "2"){
				tongbuDisableFlag = false;
			}
			if(row.connectionStatus != "Off"){
				tongbuDisableFlag = false;
				chongQidisableFlag = false;
				rizhidisableflag = false;
			}else{
				tongbuDisableFlag = true;
				chongQidisableFlag = true;
				rizhidisableflag = true;
			}

			var egwEnable = true;
			$.ajax({
				type:'POST',
				url:'${ctx}/system/egw/getEGWOperationItem.action',
				data:{egwCode: row.egwCode},
				async:false,
				dataType:'json',
				success:function(data){
					if(data) {
						egwEnable = data.egwFlag == true;
					}
				}
			});
			
			vm.menus = [
				{code: 'reboot', label: '<%=rb.getString("ChongQi")%>',row: row,disable: chongQidisableFlag},
				{code: 'sync', label: '<%=rb.getString("TongBu")%>',row: row,disable: tongbuDisableFlag},
				{code: 'collect', label:'<%=rb.getString("ShouJiBaoWen")%>',row: row},

				// {code:'info',label:'<%=rb.getString("XinXi")%>',cls: 'el-icon el-icon-operation-info',row: row},
				// {code:'setting',label:'<%=rb.getString("SheZhi")%>',cls: 'el-icon el-icon-operation-settings',row: row,show:egwEnable},
				// {code:'maintenance',label:'<%=rb.getString("Maintenance")%>',cls: 'el-icon el-icon-operation-maintenance',show:egwEnable,
				// 	child: [
				// 		{code: 'reboot', label: '<%=rb.getString("ChongQi")%>',row: row,disable: chongQidisableFlag},
				// 		{code: 'logs', label: '<%=rb.getString("RiZhi")%>',row: row,disable: rizhidisableflag},
				// 		{code: 'linkStatus', label: '<%=rb.getString("LianLuZhuangTaiChaXun")%>',row: row,},
				// 	]
				// },
				// {code:'more',label:'<%=rb.getString("GengDuoCaoZuo")%>',cls: 'el-icon el-icon-operation-more-circle',show:egwEnable,
				// 	child: [
				// 		{code: 'sync', label: '<%=rb.getString("TongBu")%>',row: row,disable: tongbuDisableFlag},
				// 	]
				// }
			];
			vm.showMenus(evt);
		},
		// 打开菜单
		showMenus(evt) {
			this.$refs.menu.show(evt)
		},
		// 关闭菜单
		hideMenus() {
			this.$refs.menu.hide()
		},
		// 菜单点击
		menuClick(menu) {
			var vm = this,
				codes = {
					info: vm.goDetail,
					setting: vm.goSetting,
					sync: vm.goSynchronize,
					reboot: vm.goReboot,
					logs: vm.goLogs,
					linkStatus:vm.queryLinkStatus,
					collect:vm.showCollectMessage,
				},
				code = menu.code;
			if(codes[code]) codes[code](menu.row);
		},
		// eGW 详情
		goDetail(row) {
			var vm = this;

			this.settingUrl = '${ctx}/egw/setting/openSettingPage.action';
			this.$refs.egwSettingPage.showSlide(function(){
				eventBus.$emit("egw-setting-init",row,'alarm')
			});
		},
		// 设置
		goSetting(row){
			var vm = this;

			vm.slideHeader = true;
			vm.slideUrl = '${ctx}/egw/pageForward/goEGWSettings.action';
			vm.slideFooter = false;
			vm.slidePosition = 'top';
			vm.slideHeight = '100%';
			vm.slideWidth = '100%';
			vm.slideTitle = '<%=rb.getString("SheZhi")%>';
			vm.settingslideCls = 'loading';
			vm.$refs.egwMonitorSlide.showSlide(()=>{
				eventBus.$emit('setting-init',row);
			});
		},
		// 同步
		goSynchronize(row){
			var vm = this,
				params={
					egwCode:row.egwCode,
				};
				
			axios.post('${ctx}/egw/monitor/refreshEGWInfo.action',stringify(params)).then(function(response){
	    		var data = response.data;
	    		if(data["success"]){
	    			vm.$refs.egwMonitorTable.refresh();
	    		}else{
	    			vm.$message.error(data["message"]) //错误提示信息
	    		}
	    	}) 
		},
		// 重启
		goReboot(row){
			var vm = this,
				urls = '${ctx}/egw/reboot/egwReboot.action',
				params = {
					egwCode: row.egwCode,
				};
			vm.$confirm('<%=rb.getString("QueDingChongQiSheBei")%>','<%=rb.getString("QueRen")%>',{
	    		customClass:'warningConfirm',
	    		confirmButtonText:'<%=rb.getString("QueDing")%>',
	    		cancelButtonText:'<%=rb.getString("QuXiao")%>',
	    		type:'warning',
	    		closeOnClickModal:false
	    	}).then(function(){
				axios.post(urls,stringify(params)).then(function(response){
					var data = response.data;
					if(data) {
						if(data["success"]){
							vm.$message({
								message: '<%=rb.getString("ChengGong")%>',
								type:'success'
							});
							vm.$refs.egwMonitorTable.refresh();
						}else{
							vm.$message.error(data["message"])
						}
					}
				}).catch(function(error){})
			});
		},
		// 日志收集
		goLogs(row) {
			var vm = this,
				params = {
					timeZone: timeZone,
					serial_number: row.egwSn,
					isReboot: 'false',
					device_code: row.egwCode,
					device_type: 'EGW',
					execute_type: 'Immediately',
					reportPeriod: '',
					start_time: undefined,
					end_time: undefined
				},
				url = '${ctx}/cell/collect/goImmediateCollectLogFile.action',
				message = '<%=rb.getString("ChengGong")%>';
		
			axios.post(url,stringify(params)).then(function(response){
				var data = response.data;
				if(data["success"]){
					vm.$message({
						message: message,
						type:'success',
					})
				}else if(data.responseCode == "901"){
					vm.$message.error(data.message);
				}else if(data.responseCode == "401"){//存在未完成的任务时，再次创建任务失败，给出提示
					vm.$message.error(data["message"]);
				}else{
					vm.$message.error('<%=rb.getString("ShouJiShiBai")%>');
				}
			})
		},
		// 链路状态查询
		queryLinkStatus(row){
			var vm = this;

			vm.slideHeader = true;
			vm.slideUrl = '${ctx}/egw/pageForward/goEGWLinkStatus.action';
			vm.slideFooter = false;
			vm.slidePosition = 'top';
			vm.slideHeight = '100%';
			vm.slideWidth = '100%';
			vm.slideTitle = '<%=rb.getString("LianLuZhuangTaiChaXun")%>';
			vm.settingslideCls = 'loading';
			vm.$refs.egwMonitorSlide.showSlide(()=>{
				eventBus.$emit('link-init',row);
			});
		},
		// 同步状态格式化
		eGWStatus(row,column,cellValue,index){

			var syncResultObj = {
				"0" : "<%=rb.getString("ChengGong")%>",
				"1" : "<%=rb.getString("ShiBai")%>",
				"2" : "<%=rb.getString("TongBuZhong")%>",
				"" : "",
			}
			return syncResultObj[cellValue];
		},
		 // table formatters
		connStatusFmt(row, value, index) {

			return connStatusFormatterSyn(value, row, index);
		},
		// 条件关闭slide事件 
		closeSlide(){
			var vm = this;
			clearInterval(egwStatusTimer);
			vm.$refs.egwMonitorSlide.hide();
		},
        // 搜索事件
        queryEgwMonitor(val){
            var vm = this;
            vm.queryParams.search_text = val;
        },
		// 监控页面  设置提交
		settingSubmit(){
			var vm = this;
		},
		settingLinkSubmit(){
			var vm = this;
			eventBus.$emit("egw-settingLink-ok");
		},
		// 关闭链路配置置页面
		closeSettingLinkSlide(){
			var vm = this;
			vm.$refs.egwSettingLinkSlide.hide();
		},
		exportEgwDevice(){
			var vm = this,
				params={},
				exportUrl ="${ctx}/egw/monitor/exportEGWToCSV.action";
	
			params.search_text = vm.queryParams.search_text;
			exportByForm(exportUrl,params);
		},
		// 打开链路配置
		openLinkSetting(opType,row,deviceType){
			var vm = this,
				openLinkType = opType;
			
			vm.settingLinkSlideHeader = true;
			vm.settingLinkSlideUrl = '${ctx}/egw/pageForward/goEGWSettingsEnbLink.action';
			vm.settingLinkSlideFooter = true;
			vm.settingLinkSlidePosition = 'top';
			vm.settingLinkSlideHeight = '100%';
			vm.settingLinkSlideWidth = '100%';
			vm.settingLinkSlideTitle = openLinkType;
			vm.$refs.egwSettingLinkSlide.showSlide(()=>{
				eventBus.$emit("link-init",openLinkType,row,deviceType);
			});
		},
		// 关闭链路配置
		closeLinkSetting(){
			var vm = this;
			vm.$refs.egwSettingLinkSlide.hide();
		},
		// 名称下发
		syncName(code) {
			var vm = this,
				urls = '${ctx}/egw/monitor/syncEgwName.action'
				params={
					egwCode:code
				};

			axios.post(urls,stringify(params)).then(function(response){
				var data = response.data;
				if(data["success"]){
					vm.$message({
						message: '<%=rb.getString("ChengGong")%>',
						type:'success',
					})
					vm.$refs.egwMonitorTable.refresh();
					vm.closeSyncName();
				}else{
					vm.$message.error('<%=rb.getString("ShiBai")%>');
				}
			})
		},
		closeSyncName(){
			document.body.click();
		},
		openSettingPage(row){
			this.settingUrl = '${ctx}/egw/setting/openSettingPage.action';
			this.$refs.egwSettingPage.showSlide(function(){
				eventBus.$emit("egw-setting-init",row,'')
			});
		},
		closeSettingPage(){
			this.$refs.egwSettingPage.hide();
		},
		// 打开收集报文弹窗
		showCollectMessage(row) {
			var vm = this,
				paramsExist = {
					type: 'wcg',
					operatorCode: operatorCodeGloab
				};

			axios.post('${ctx}/trace/isExistTracingDevice.action', stringify(paramsExist)).then(function(res){
				var data = res.data;

				if(data && data.isExist == 'true') {
					vm.collectExisted = true;
					vm.existedMsgSN = data.serialNumber;
				}else {
					vm.collectExisted = false;
					vm.existedMsgSN = '';
				}
			});

			vm.collectForm.collectInterval = '10';
			vm.collectMessageShow = true;
		},
		// 确认收集报文
		sendCollect() {
			var vm = this,
				row = vm.rowData || {},
				time = vm.collectForm.collectInterval+':00',
				params = {
					deviceCode: row.egwCode,
					serialNumber: row.egwSn,
					type: 'wcg',
					operatorCode: operatorCodeGloab,
					collectInterval: vm.collectForm.collectInterval
				},
				paramsExist = {
					type: 'wcg',
					operatorCode: operatorCodeGloab
				};

			axios.post('${ctx}/trace/isExistTracingDevice.action', stringify(paramsExist)).then(function(res){
				var data = res.data;

				if(data && data.isExist == 'true') {
					vm.$message.error('SN=' + data.serialNumber + '<%=rb.getString("ZhengZaiShouJi")%>');
				}else {
					axios.post('${ctx}/trace/start.action', stringify(params)).then(function(res){
						var data = res.data;

						if(data.success == true) {
							vm.startCollectInterval(time);
							vm.queryLatestCollectInfo();
							vm.collectMessageShow = false;
							
							vm.$message.success('<%=rb.getString("ChengGong")%>');
						}else {
							vm.$message.error(data.message);
						}
					});
				}
			});
		},
		// 查询最新收集报文详情
		queryLatestCollectInfo() {
			var vm = this,
				params = {
					type: 'wcg',
					operatorCode: operatorCodeGloab
				};

			axios.post('${ctx}/trace/queryLatestMessageTraceDeviceInfo.action', stringify(params)).then(function(res){
				var data = res.data;

				if(data) {
					vm.collectActiveSn = data.serialNumber;
					vm.collectDeviceCode = data.deviceCode;

					if(data.status == '0' && data.remainTime) {
						vm.startCollectInterval(data.remainTime);
					}else if(data.status == '1'){
						vm.collectTaskTime = '';
						vm.startCollectInterval('00:01');
					}

					if(['',undefined].includes(data.status) && ['',undefined].includes(data.serialNumber)) {
						vm.isEgwCollectExisted = false;
					}else {
						vm.isEgwCollectExisted = true;
					}
				}
			});
		},
		// 查看收集报文详情
		viewMsg() {
			var vm = this,
				params = {
					serialNumber: vm.collectActiveSn,
					type: 'wcg'
				};
				   
			vm.collectInfoShow = true;

			axios.post('${ctx}/trace/queryMessageTraceInfo.action', stringify(params)).then(function(res){
				var data = res.data;

				if(data) {
					vm.collectContent = data;
					vm.collectInfoShow = true;
				}
			});
		},
		// 下载收集报文
		downloadMsg() {
			var vm = this,
				params = {
					serialNumber: vm.collectActiveSn,
					type: 'wcg',
					timeZone: timeZone
				};

			exportByForm('${ctx}/trace/downLoadMessageTraceInfo.action',params);
		},
		// 清除收集报文任务
		clearMsg() {
			var vm = this,
				params = {
					deviceCode: vm.collectDeviceCode,
					serialNumber: vm.collectActiveSn,
					type: 'wcg',
					operatorCode: operatorCodeGloab
				};

			axios.post('${ctx}/trace/clear.action', stringify(params)).then(function(res){
				var data = res.data;

				if(data.success == true) {
					vm.$message.success('<%=rb.getString("ChengGong")%>');
					vm.queryLatestCollectInfo();
				}else {
					vm.$message.error(data.message);
				}
			});
		},
		// 停止收集报文
		stopCollect() {
			var vm = this,
				params = {
					deviceCode: vm.collectDeviceCode,
					serialNumber: vm.collectActiveSn,
					type: 'wcg',
					operatorCode: operatorCodeGloab
				};

			axios.post('${ctx}/trace/stop.action', stringify(params)).then(function(res){
				var data = res.data;

				if(data.success == true) {
					vm.$message.success('<%=rb.getString("ChengGong")%>');
					vm.queryLatestCollectInfo();
				}else {
					vm.$message.error(data.message);
				}
			});
		},
		// 定时刷新收集报文任务
		startCollectInterval(time) {
			var vm = this,
				list = time.split(':'),
				totalTime = list[0]*60 + list[1]*1;

			clearInterval(egwCollectInterval);
			
			egwCollectInterval = setInterval(function() {
				totalTime -= 1;
				vm.collectTaskTime = vm.formaterTime(Math.floor(totalTime/60)) + ':' + vm.formaterTime(totalTime%60);

				if(totalTime<1) {
					clearInterval(egwCollectInterval);
					vm.collectTaskTime = '';
				}
			},1000);
		},
		formaterTime(num) {

			return num < 10? '0'+num : num;
		},
	},
	mounted(){
		eventBus.$off('close-setting').$on('close-setting',this.closeSlide);
		eventBus.$off('open-linkSetting').$on('open-linkSetting',this.openLinkSetting);
		eventBus.$off('close-linkSetting').$on('close-linkSetting',this.closeLinkSetting);
		eventBus.$off('cancel-egw-setting').$on('cancel-egw-setting',this.closeSettingPage);
	}
	
})

</script> 
