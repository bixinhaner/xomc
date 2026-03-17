<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	#egwMonitorInfo .InfoContent{
		padding-left: 20px;
		padding-top: 20px;
	}
	#egwMonitorInfo .InfoContent .InfoTitleBoxCls{
		height: 60px;
		box-sizing: border-box;
		display: flex;
	}
	#egwMonitorInfo .InfoTitleBoxCls .titleImgBoxCls{
		padding: 20px 0;
		border: 1px solid #e9e9e9;
		border-radius: 10px;
		box-shadow: 1px 2px 10px rgba(0,0,0,0.2);
	}
	#egwMonitorInfo .InfoTitleBoxCls .titleTextBoxCls{
		font-weight: bold; 
		color: #666;
		padding: 0px 20px;
	}
	#egwMonitorInfo .InfoContent .InfoMainBoxCls{
		width: 100%;
		display: flex;
	}
	#egwMonitorInfo .InfoMainBoxCls .MainBoxLeftCls{
		padding-top: 30px;
		width:  calc(100% - 460px)!important;
		min-width:900px;
	}
	#egwMonitorInfo .InfoMainBoxCls .MainBoxRightCls{
		padding-left: 20px;
		width: 460px;
	}
	#egwMonitorInfo .MainBoxLeftCls .leftHeaderBoxCls{
		display: flex;
		height: 200px;
		width: 100%;
		margin-bottom: 20px;
	}
	#egwMonitorInfo .leftHeaderBoxCls .basicInfoBoxCls{
		width: calc(100% - 380px)!important;
		height: 200px;
		border: 1px solid #E9E9E9;
		border-top: 2px solid #4D84FF;
		border-radius: 3px;
		padding-left: 20px;
		margin-right: 20px;
	}
	#egwMonitorInfo .leftHeaderBoxCls .actionListBoxCls{
		width: 380px;
		height: 200px;
		border: 1px solid #E9E9E9;
		border-top: 2px solid #4D84FF;
		border-radius: 3px;
		padding-left: 20px;
	}
	#egwMonitorInfo .basicInfoBoxCls .basicInfoTitleCls, #egwMonitorInfo .actionListBoxCls .actionListTitleCls{
		font-size: 14px;
		font-weight: bold;
		color: #333;
		padding: 20px 0px;
	}
	#egwMonitorInfo .basicInfoBoxCls .basicInfoMainCls{
		display: flex;
		flex-wrap: wrap;
		width: 100%;
	}
	#egwMonitorInfo .basicInfoMainCls .basicInfoItemCls{
		width: 50%;
		margin-bottom: 14px;
	}
	#egwMonitorInfo .basicInfoItemCls span:first-child{
		display: inline-block;
		width: 126px;
		font-size: 12px;
		color: #999999;
		font-weight: 400;
	}
	#egwMonitorInfo .basicInfoItemCls span:last-child{
		font-size: 12px;
		color: #333333;
		font-weight: 400;
	}
	#egwMonitorInfo .actionListMainCls .syncBoxCls{
		height: 70px;
		display: flex;
		align-items: center
	}
	#egwMonitorInfo .actionListMainCls .rebootBoxCls{
		border-top: 1px solid #E9E9E9;
		height: 70px;
		display: flex;
		align-items: center
	}
	#egwMonitorInfo .actionListMainCls .pro-class{
		width:100px;
		height:6px;
		border-radius:10px;
		margin-left: 20px;
		margin-right: 50px;
	}
	#egwMonitorInfo .actionListMainCls .pro-label{
		width: 80px;
		font-size:12px;
		color:#333;
	}
	#egwMonitorInfo .actionListMainCls .pro-wait{
		background:#DCDFE6;
	}
	#egwMonitorInfo  .pro-run{
		background: url('${ctx}/css/images/bi/enb_progress.gif') no-repeat center;
	}
	#egwMonitorInfo .actionListMainCls .pro-suc{
		background:#67D972;
	}
	#egwMonitorInfo .actionListMainCls .pro-fail{
		background:#E88282;
	}
	#egwMonitorInfo .MainBoxLeftCls .leftAlarmBoxCls{
		margin-bottom: 20px;
	}
	#egwMonitorInfo .leftAlarmBoxCls .alarmHerderCls{
		height: 26px;
		display: flex;
		justify-content: space-between;
		margin-bottom: 10px;
	}
	#egwMonitorInfo .leftAlarmBoxCls .alarmTableBoxCls,#egwMonitorInfo .leftLogBoxCls .logTableBoxCls{
		height: 228px;
		width: 100%;
		border: 1px solid #E9E9E9;
		box-sizing: border-box;
	}
	.egwAlarmMinor,.egwAlarmMajor,.egwAlarmCritical,.egwAlarmWarning{
		display: flex;
		align-items: center;
	}
	.egwAlarmMinor .el-icon:before{
		color: #FFDA41;
		font-size: 20px;
	}
	.egwAlarmMajor .el-icon:before{
		color: #FF973E;
		font-size: 20px;
	}
	.egwAlarmCritical .el-icon:before{
		color: #FC5959;
		font-size: 20px;
	}
	.egwAlarmWarning .el-icon:before{
		color: #60BEFC;
		font-size: 20px;
	}
	.egwUnconfirmInactive,.egwConfirmInactive,.egwUnconfirmActive,.egwConfirmActive{
		display: flex;
		align-items: center;
	}
	.egwUnconfirmInactive .el-icon:before{
		color: #E88282;
		font-size: 20px;
	}
	.egwConfirmInactive .el-icon:before{
		color: #E88282;
		font-size: 20px;
	}
	.egwUnconfirmActive .el-icon:before{
		color: #67D972;
		font-size: 20px;
	}
	.egwConfirmActive .el-icon:before{
		color: #67D972;
		font-size: 20px;
	}
	.AlarmInfoDialogCls .el-dialog__body{
		height: 600px;
	}
	#egwMonitorInfo .leftLogBoxCls{
		margin-bottom: 20px;
	}
	#egwMonitorInfo .leftLogBoxCls .logHerderCls{
		height: 26px;
		display: flex;
		margin-bottom: 10px;
	}
	#egwMonitorInfo .logHerderCls .el-button:hover .el-icon:before{
		color: #FFFFFF;
	}

	#egwMonitorInfo .flex-col-chart .el-card, #egwMonitorInfo .flex-col-chart{
		overflow: unset;
	}
	#egwMonitorInfo .flex-col-chart {
		position: relative;
		flex: 1 auto;
		overflow: hidden;
	}
	#egwMonitorInfo .half-persent {
		height: calc(50% - 10px);
		margin: 0px 0px 20px 0px;
		flex: 1;
		background-color: #fff;
		min-height: 240px;
		min-width:400px;
		border:1px solid #E9E9E9;
	}
	#egwMonitorInfo .el-card__footer{
		 height:unset;
		 line-height: unset;
		 padding-left:unset; 
		 border:none;
	 }
	#egwMonitorInfo .MainBoxRightCls .rightTitleBoxCls{
		height: 30px;
		color: #333333;
		font-weight: bold;
	}
	#egwMonitorInfo .MainBoxRightCls .rightChartBoxCls{
		display: flex;
		flex-direction: column;
	}
	#egwMonitorInfo .herderButtonBoxCls{
		display: flex;
	}
	#egwMonitorInfo .herderButtonBoxCls > div{
		height: 26px;
		padding: 0px 10px;
		box-sizing: border-box;
		border-radius: 2px;
		font-size: 12px;
		font-weight: 500;
		display: flex;
		align-items: center;
		cursor: pointer;
	}
	#egwMonitorInfo .herderButtonBoxCls .buttonDefaultCls{
		border:1px solid #E9E9E9;
		color: #363B4E;
	}
	#egwMonitorInfo .herderButtonBoxCls .buttonSelectCls{
		border:1px solid var(--main-color)!important;
		color: var(--main-color);
	}
	#egwMonitorInfo .leftChartBoxCls{
		display: flex;
		flex-wrap: wrap;
	}
</style>
<div class="panelDefault" id="egwMonitorInfo" style="overflow:auto">
	<div class="InfoContent">
		<div class="InfoTitleBoxCls">
			<!--<div class="titleImgBoxCls">
				<img src="${ctx}/skin/BaiCells/images/login/logo_about_new.png" style="width: 60px;"/>
			</div>-->
			<div class="titleTextBoxCls">
				<div style="padding:10px 0px;"><%=rb.getString("EGWMingCheng")%><%=rb.getString("MaoHao")%> {{deviceData.egwName}}</div>
				<div><%=rb.getString("eGWBianMa")%><%=rb.getString("MaoHao")%> {{deviceData.egwSn}}</div>
			</div>
		</div>
		<div class="InfoMainBoxCls">
			<div class="MainBoxLeftCls">
				<div class="leftHeaderBoxCls">
					<div class="basicInfoBoxCls">
						<div class="basicInfoTitleCls">Basic Information</div>
						<div class="basicInfoMainCls">
							<div class="basicInfoItemCls">
								<span>eNB Number</span>
								<span>{{deviceData.enbNumber}}</span>
							</div>
							<div class="basicInfoItemCls">
								<span>UE Number</span>
								<span>{{deviceData.ueCount}}</span>
							</div>
							<div class="basicInfoItemCls">
								<span>UPLink Traffic</span>
								<span>{{deviceData.uplink_traffic}}MB</span>
							</div>
							<div class="basicInfoItemCls">
								<span>DownLink Traffic</span>
								<span>{{deviceData.downlink_traffic}}MB</span>
							</div>
							<div class="basicInfoItemCls">
								<span>eGW IP</span>
								<span>{{deviceData.egwIp}}</span>
							</div>
							<div class="basicInfoItemCls">
								<span>eGW Port</span>
								<span>{{deviceData.egwPort}}</span>
							</div>
							<div class="basicInfoItemCls">
								<span><%=rb.getString("ZhuBeiZhuangTai")%></span>
								<span>{{deviceData.ha_status}}</span>
							</div>
							<div class="basicInfoItemCls">
								<span>Software Version</span>
								<span>{{deviceData.softwareVersion}}</span>
							</div>
						</div>
					</div>
					<div class="actionListBoxCls">
						<div class="actionListTitleCls">Action List</div>
						<div class="actionListMainCls">
							<div class="syncBoxCls">
								<span class='pro-label'><%=rb.getString("TongBu")%></span>
								<p class='pro-class' :class="synClass"></p>
								<el-button type="primary" size="mini" class="CODE_EGW hidden" :disabled="actionListButtonDis" @click="changeSynStatus"><%=rb.getString("YingYong")%></el-button>
							</div>
							<div class="rebootBoxCls">
								<span class='pro-label'><%=rb.getString("ChongQi")%></span>
								<p class='pro-class' :class="rebootClass"></p>
								<el-button type="primary" size="mini" class="CODE_EGW hidden" :disabled="actionListButtonDis" @click="changeRebootStatus"><%=rb.getString("YingYong")%></el-button>
							</div>
						</div>
					</div>
				</div>
				<div class="leftAlarmBoxCls">
					<div class="alarmHerderCls">
						<div style="font-size: 14px;font-weight: bold;">Alarm</div>
						<div class="herderButtonBoxCls">
							<div :class="alarmType == 'ACTIVE' ? 'buttonSelectCls': 'buttonDefaultCls'" @click="alarmTypeChange('ACTIVE')"><%=rb.getString("HuoDongGaoJing")%></div>
							<div :class="alarmType == 'HISTORY' ? 'buttonSelectCls': 'buttonDefaultCls'" @click="alarmTypeChange('HISTORY')"><%=rb.getString("LiShiGaoJing")%></div>
						</div>
					</div>
					<div class="alarmTableBoxCls">
						<!--:url="alarmTableUrl   :data="data"" -->
						<el-ctable ref="alarmTable" :rownumber="true" :time="6" id="alarmTable" :url="alarmTableUrl" :query-params="alarmQueryParams" height="100%" pagination="true">
							<el-table-column label='' width="30" prop="">
								<template slot-scope="scope">
										<div class="el-icon el-icon-operation-more" @click="optAlarmClick(scope.row,event)" v-clickoutside="handerClose" style="cursor: pointer;"></div>
								</template>
							</el-table-column>
							<el-table-column label='<%=rb.getString("GaoJingJiBie")%>' min-width="100"  prop="alarm_serverity_value" sortable>
								<template slot-scope="scope">
									<div v-if="scope.row.alarm_serverity_value == 'Minor'" class="egwAlarmMinor">
										<span class="el-icon el-icon-status-alarm" style="margin-right:5px;"></span><%=rb.getString("CiYaoGaoJing")%>
									</div>
									<div v-else-if="scope.row.alarm_serverity_value == 'Major'"  class="egwAlarmMajor">
										<span class="el-icon el-icon-status-alarm" style="margin-right:5px;"></span><%=rb.getString("ZhuYaoGaoJing")%>
									</div>
									<div v-else-if="scope.row.alarm_serverity_value == 'Critical'" class="egwAlarmCritical">
										<span class="el-icon el-icon-status-alarm" style="margin-right:5px;"></span><%=rb.getString("JinJiGaoJing")%>
									</div>
									<div v-else-if="scope.row.alarm_serverity_value == 'Warning'" class="egwAlarmWarning">
										<span class="el-icon el-icon-status-alarm" style="margin-right:5px;"></span><%=rb.getString("JingGaoGaoJing")%>
									</div>
								</template>
							</el-table-column>
							<el-table-column label='<%=rb.getString("GaoJingWeiYiBiaoZhi")%>' min-width="130" prop="alarm_identifier"></el-table-column>
							<el-table-column label='<%=rb.getString("KeNengYuanYin")%>' min-width="180" prop="alarm_name" show-overflow-tooltip></el-table-column>
							<el-table-column label='<%=rb.getString("GaoJingZhuangTai")%>' min-width="190" prop="deal_state"  show-overflow-tooltip sortable>
								<template slot-scope="scope">
									<div v-if="scope.row.deal_state == '0'" class="egwUnconfirmInactive">
										<span class="el-icon el-icon-status-unconfirmActive" style="margin-right:5px;"></span><%=rb.getString("WeiQueRenWeiQingChu")%>
									</div>
									<div v-else-if="scope.row.deal_state == '1'" class="egwConfirmInactive">
										<span class="el-icon el-icon-status-confirmActive" style="margin-right:5px;"></span><%=rb.getString("YiQueRenWeiQingChu")%>
									</div>
									<div v-if="scope.row.deal_state == '2'" class="egwUnconfirmActive">
										<span class="el-icon el-icon-status-unconfirmActive" style="margin-right:5px;"></span><%=rb.getString("WeiQueRenYiQingChu")%>
									</div>
									<div v-else-if="scope.row.deal_state == '3'" class="egwConfirmActive">
										<span class="el-icon el-icon-status-confirmActive" style="margin-right:5px;"></span><%=rb.getString("YiQueRenYiQingChu")%>
									</div>
								</template>
							</el-table-column>
							<el-table-column label='<%=rb.getString("GuZhangShiJian")%>' min-width="150" prop="event_time" sortable></el-table-column>
							<el-table-column v-if="alarmType == 'ACTIVE'" label='<%=rb.getString("GengXinShiJian")%>' width="150"  prop="upd_time" sortable></el-table-column>
							<el-table-column v-if="alarmType == 'HISTORY'" label='<%=rb.getString("GaoJingQingChuShiJian")%>' width="150"  prop="clear_time" sortable></el-table-column>
							<el-table-column label='<%=rb.getString("GaoJingCiShu")%>'  min-width="100" prop="alarm_count" sortable></el-table-column>
						</el-ctable>
						<el-cmenu ref="menusAlarm" :data="menusAlarm" @click="clickAlarmMenu"></el-cmenu>
					</div>
					
				</div>
				<div class="leftLogBoxCls">
					<div class="logHerderCls">
						<div style="font-size: 14px;font-weight: bold;">
							Logs
							<el-button  @click="collectLogsClick" type="primary" plain style="margin-left:20px;">
								<i class="el-icon el-icon-collectLogs"></i>
								<span>Collect Logs</span>
							</el-button>
						</div>
					</div>
					<div class="logTableBoxCls">
						<!--:url="logTableUrl" :data="logData" -->
						<el-ctable ref="logTable" :rownumber="true" id="logTable" :time="6" :url="logTableUrl" :query-params="logQueryParams" height="100%" pagination="true">
							<el-table-column label='' width="30" prop="">
								<template slot-scope="scope">
										<div class="el-icon el-icon-operation-more" @click="optLogClick(scope.row,event)" v-clickoutside="handerClose" style="cursor: pointer;"></div>
								</template>
							</el-table-column>
							<el-table-column label='<%=rb.getString("eGWBianMa")%>' min-width="150" prop="serial_number"></el-table-column>
							<el-table-column label='<%=rb.getString("ShouJiZhuangTai")%>' min-width="150" prop="task_status">
								<template slot-scope="scope">
									<!-- 0 等待 ：可终止 ，可删除-->					
									<div v-if="scope.row.task_status == 0">
										<span class="el-icon el-icon-status-waiting1"></span>
										<span style="margin-left:10px;"><%=rb.getString("DengDai")%></span>
									</div>
									<!--1 进行中：可终止，不可删除-->
									<div v-if="scope.row.task_status == 1">
										<span style="display:inline-block;width:60px;height:6px;border-radius:10px;" class="pro-run"></span>
										<span style="margin-left:10px;"><%=rb.getString("JinXingZhong")%></span>
									</div>
									<!-- 2 成功-->
									<div v-if="scope.row.task_status == 2">
										<span class="el-icon el-icon-status-success"></span>
										<span style="margin-left:10px;"><%=rb.getString("ChengGong")%></span>
									</div>
									<!-- 3 失败-->
									<div v-if="scope.row.task_status == 3">
										<span class="el-icon el-icon-status-failed"></span>
										<span style="margin-left:10px;"><%=rb.getString("LogsShiBai")%></span>
									</div>
									<!-- 4 终止-->
									<div v-if="scope.row.task_status == 4">
										<span class="el-icon el-icon-status-terminate"></span>
										<span style="margin-left:10px;"><%=rb.getString("ZhongZhi")%></span>
									</div>
								</template>
							</el-table-column>
							<el-table-column label='<%=rb.getString("ShiBaiYuanYin")%>' min-width="200" prop="failureReason" show-overflow-tooltip="true"></el-table-column>
							<el-table-column label='<%=rb.getString("GengXinShiJian")%>' min-width="150" prop="update_time"></el-table-column>
						</el-ctable>
						<el-cmenu ref="menusLog" :data="menusLog" @click="clickLogMenu"></el-cmenu>
					</div>
				</div>
				<div class="leftChartBoxCls">
					<div :class="chartClass">
						<el-card shadow="hover" class="chart-card">
							<div id="cpuChart" style="width: 400px;height: 270px;"></div>
						</el-card>
					</div>
					<div :class="chartClass" style="margin:0px 20px;">
						<el-card shadow="hover" class="chart-card">
							<div id="memoryChart" style="width: 400px;height: 270px;"></div>
						</el-card>
					</div>
					<div :class="chartClass">
						<el-card shadow="hover" class="chart-card">
							<div id="diskChart" style="width: 100%;height: 270px;"></div>
						</el-card>
					</div>
				</div>
			</div>
			<div class="MainBoxRightCls">
				<div class="rightTitleBoxCls">History</div>
				<div class="rightChartBoxCls">
					<div :class="chartClass">
						<el-card shadow="hover" class="chart-card">
							<div id="ueCountChart" style="width: 460px;height: 270px;"></div>
						</el-card>
					</div>
					<div :class="chartClass">
						<div class="herderButtonBoxCls" style="position:absolute;right:20px;top:10px;z-index:999">
							<div :class="upLinkUnitType == 'MB' ? 'buttonSelectCls': 'buttonDefaultCls'" @click="upLinkUnitTypeChange('MB')">MB</div>
							<div :class="upLinkUnitType == 'GB' ? 'buttonSelectCls': 'buttonDefaultCls'" style="border-left:1px solid #FFF;" @click="upLinkUnitTypeChange('GB')">GB</div>
						</div>
						<el-card shadow="hover" class="chart-card" style="position:relative;">
							<div id="upLinkChart" style="width: 460px;height: 270px;" ></div>
						</el-card>
					</div>
					<div :class="chartClass">
						<div class="herderButtonBoxCls" style="position:absolute;right:20px;top:10px;z-index:999">
							<div :class="downLinkUnitType == 'MB' ? 'buttonSelectCls': 'buttonDefaultCls'" @click="downLinkUnitTypeChange('MB')">MB</div>
							<div :class="downLinkUnitType == 'GB' ? 'buttonSelectCls': 'buttonDefaultCls'" style="border-left:1px solid #FFF;" @click="downLinkUnitTypeChange('GB')">GB</div>
						</div>
						<el-card shadow="hover" class="chart-card" style="position:relative;">
							<div id="downLinkChart" style="width: 460px;height: 270px;" ></div>
						</el-card>
					</div>
				</div>
			</div>
		</div>
	</div>
	<!--告警详情弹窗-->
	<el-dialog :title="'<%=rb.getString("XiangXiXinXi")%>'" :visible.sync="showAlarmInfo" ref="AlarmInfoDialog" class="AlarmInfoDialogCls" :width="alarmInfoDialogWidth"
		:close-on-click-modal="false" :url="alarmInfoDialogUrl" @close="closeAlarmInfoDialog" @success="openAlarmInfoDialogSuc" :append-to-body="true"></el-dialog>
	<!--告警清除弹窗-->
	<el-dialog :title="'<%=rb.getString("QueRen")%>'" :visible.sync="showClearAlarmDialog" width="500" 
		:close-on-click-modal="false" top="15vh" @close="clearAlarmDialogClose" :append-to-body="true">
		<el-form  :model="clearForm" ref="clearForm" label-position="top">
			<el-form-item>
				<%=rb.getString("QueRenQingChuGaoJing")%>
			</el-form-item>
			<el-form-item label="<%=rb.getString("MiaoShu")%>" prop="description">
				<el-input type="textarea" :rows="3" v-model="clearForm.description">
			</el-form-item>
		</el-form>
		<span slot="footer" class="dialog-footer">
			<div class="buttonGroup">
				<el-button type="primary" @click="clearAlarmSubmit"><%=rb.getString("QueDing")%></el-button>
				<el-button @click="clearAlarmDialogClose"><%=rb.getString("QuXiao")%></el-button>
			</div>	
		</span>
	</el-dialog>
	<!--告警确认弹窗-->
	<el-dialog :title="'<%=rb.getString("QueRenGaoJing")%>'" :visible.sync="showConfirmAlarmDialog" width="500" 
		:close-on-click-modal="false" top="15vh" @close="confirmAlarmDialogClose" :append-to-body="true">
		<el-form :model="confirmForm" ref="confirmForm" label-position="top">
			<el-form-item v-if="confirmFlag" label="<%=rb.getString("QueRenRen")%>" prop="confirmUser">
				<el-input :disabled="true" v-model="confirmForm.confirmUser"></el-input>
			</el-form-item>
			<el-form-item v-if="confirmFlag" label="<%=rb.getString("QueRenShiJian")%>" prop="confirmTime">
				<el-input :disabled="true" v-model="confirmForm.confirmTime">
			</el-form-item>
			<el-form-item label="<%=rb.getString("MiaoShu")%>" prop="description">
				<el-input type="textarea" :rows="3" v-model="confirmForm.description">
			</el-form-item>
		</el-form>
		<span slot="footer" class="dialog-footer">
			<div class="buttonGroup">
				<el-button type="primary" @click="confirmAlarmSubmit"><%=rb.getString("QueDing")%></el-button>
				<el-button @click="confirmAlarmDialogClose"><%=rb.getString("QuXiao")%></el-button>
			</div>	
		</span>
	</el-dialog>
</div>

<script type="text/javascript">
	/**
	*  页面编辑和只读模式通过readonly控制
	*  校验规则也由readonly决定
	**/
	
	var timeParam = getNowTimeToZoneTimeRange(timeZone, 144);
	var start_time = timeParam.start_time.substring(0,11)+"00:00:00";
	var end_time = timeParam.end_time;
	var egwMonitorInfoVue = new Vue({
		el: '#egwMonitorInfo',
		data(){
			var vm = this;
			return {
				egwCode:'',
				deviceData:{},
				chartList: ['ueCountChart','upLinkChart','downLinkChart','cpuChart','memoryChart','diskChart'],
				charts: {},
				rowDataAlarm:'',
				alarmType:'ACTIVE',
				alarmTableUrl:'',
				alarmQueryParams:{
					timeZone:timeZone,
					alarmType:'ACTIVE',
					queryType:'View',
					alarmServerity:'31001,31002,31003,31004',
					deviceCode:'',
					neType:'EGW'
				},
				alarmInfoDialogUrl:'',
				showAlarmInfo:false,
				alarmInfoDialogWidth:'850px',
				menusAlarm:[],
				logTableUrl:'',
				rowDataLog:'',
				logQueryParams:{
					timeZone:timeZone,
					device_code:'',
					device_type:'EGW'
				},
				menusLog:[],
				synClass:'pro-run',
				rebootClass:'pro-wait',
				data:[
					{
						additional_information: "",
						additional_text: "",
						alarm_cn_name: "eNB网络断开",
						alarm_count: 1,
						alarm_en_name: "eNB Disconnected",
						alarm_id: 4046,
						alarm_identifier: "7",
						alarm_name: "eNB Disconnected",
						alarm_serverity: 31001,
						alarm_serverity_value: "Critical",
						alarm_type: "history",
						clear_time: "",
						cn_propable_cause: "基站与网管之间连线断开",
						deal_state: 0,
						en_propable_cause: "The connection between the base station and the OMC is disconnected.Network failure between base station and OMC.",
						equip_info: "SN=120200010817BAP0042;CellName=unknown name",
						event_time: "2021-03-25 16:01:55",
						event_type: "30000",
						event_type_value: "Communication Alarm",
						ne_type: "ENB",
						propable_cause: "The connection between the base station and the OMC is disconnected.Network failure between base station and OMC.",
						serial_number: "120200010817BAP0042",
						small_cell_code: "48BF74_120200010817BAP0042",
						specific_problem: "Smallcell is disconnected.",
						upd_time: ""
					}
				],
				logData:[{
					device_code:'112205_p1214568205635825',
					device_type:'eGW',
					end_time:'2021-06-23 13:56:00',
					execute_type:'Immediately',
					file_num:5,
					progress_detail:'ShouJiWanCheng',
					report_period:'',
					serial_number:'p1214568205635825',
					start_time:'2021-06-23 13:56:00',
					task_id:'e12353254154S1D5325W321S5W',
					task_status:1,
					update_time:'2021-06-23 13:55:00'
				}],
				showClearAlarmDialog:false,
				clearForm:{
					description:'',
				},
				showConfirmAlarmDialog:false,
				confirmFlag:false,
				confirmForm:{
					confirmUser:'',
					confirmTime:'',
					description:'',
				},
				currenUpLinkIndex:6,
				currenDownLinkIndex:6,
				upLinkUnitType:'MB',
				downLinkUnitType:'MB'
			}
		},
		computed: {
			// 图表样式
			chartClass(){
				return {
					'flex-col-chart': true,
					'half-persent': true
				};
			},
			actionListButtonDis(){

				return  this.deviceData.connectionStatus? (this.deviceData.connectionStatus == 'Off' ? true : false) : true ;
			}
		},
		watch: {},
		methods: {
			// 图表初始化
			init(id){
				var vm = this;
				
				vm.egwCode = id;
				vm.getDeviceDetail(id)
				vm.chartList.map(function(code){
					var chart = document.querySelector('#'+code);
					if(chart) {
						vm.charts[code] = echarts.init(chart);
						
						// 初始化时间轴切换事件
						vm.charts[code].on('timelinechanged',function(p){
							vm.reloadChart(code,p.currentIndex);
							if(code == 'upLinkChart'){
								vm.currenUpLinkIndex = p.currentIndex;
							}else if(code == 'downLinkChart'){
								vm.currenDownLinkIndex = p.currentIndex;
							}
						});
					}
					
					// 窗口缩放时自适应
					window.removeEventListener('resize',vm.resizeChart);
					window.addEventListener('resize',vm.resizeChart);

					vm.reloadChart(code,6);
				});
			},
			// 设备详情信息查看
			getDeviceDetail(id){
				var vm = this,
					params={
						egwCode:id
					};

				axios.post('${ctx}/egw/monitor/getBasicInfo.action',stringify(params)).then(function(response){
					var data = response.data;
					vm.deviceData = data;
					vm.alarmQueryParams.deviceCode = vm.deviceData.egwCode;
					vm.logQueryParams.device_code = vm.deviceData.egwCode;
					vm.alarmTableUrl = '${ctx}/fault/view/queryViewPageList.action';
					vm.logTableUrl = '${ctx}/cell/collect/getImmediateCollectLogTaskPageList.action';
					vm.checkStatus();
					egwMonitor.settingslideCls = '';
				}) 
			},
			// 更新重启 、 同步状态
			checkStatus(){
				var vm = this,
					urls="${ctx}/egw/monitor/getSyncRebootStatus.action",
					params = {
						egwSn: vm.deviceData.egwSn,
						egwCode: vm.deviceData.egwCode,
					};
				var dom = $("#egwMonitorInfo");
				if(dom.length == 0) clearInterval(egwStatusTimer);
				axios.post(urls,stringify(params)).then(function(response){
					var data = response.data;
					var rebootStatus = data.rebootStatus;
					var syncStatus = data.syncStatus;
					if(rebootStatus == '1') vm.rebootClass = 'pro-wait';//未进行
					if(rebootStatus == '2') vm.rebootClass = 'pro-run';//进行中
					if(rebootStatus == '3') vm.rebootClass = 'pro-fail';//失败
					if(rebootStatus == '4') vm.rebootClass = 'pro-suc';//成功
					if(syncStatus == "updating") vm.synClass = 'pro-run';
					if(syncStatus == "On") vm.synClass = 'pro-suc';
					if(syncStatus == "Off") vm.synClass = 'pro-wait';
					if(syncStatus == "Exception") vm.synClass = 'pro-fail';
				})
			},
			// 所有图表自适应
			resizeChart(){
				var vm = this;
				vm.chartList.map(function(code){
					if(vm.charts[code]) vm.charts[code].resize();
				});
			},
			/**
			* 指定图表数据刷新
			* @param code{string} 图表类型
			* @param index{number}  时间轴下标
			*/
			reloadChart(code,index){
				var vm = this;
				
				if(vm.charts[code]) {
					if(['ueCountChart','upLinkChart','downLinkChart'].includes(code)) {
						vm.proccessChartData(code,index);// 统计数据刷新
					}
					if(['cpuChart','memoryChart','diskChart'].includes(code)) {
						vm.getDeviceChartData(code,index);// 统计数据刷新
					}
				}
			},
			/**
			* 统计数据刷新 UE数 上下行流量
			* @param code{string} 图表类型
			* @param index{number}  时间轴下标
			*/
			proccessChartData(code,index){
				var vm = this,
					s_time = getYesterDay(6-index).substring(0,10)+' 00:00:00',
					e_time = getYesterDay(6-index).substring(0,10)+' 23:59:59';
				
				if(index==6) {
					e_time = end_time;
				}else{
					e_time = getYesterDay(6-index-1).substring(0,10)+' 00:00:00';
				}
				var chartData={
					legendNames: [],
					data: [],
					yAxisName: '',
					xAxisData:[],
					color:[],
					pointerCount: 6,
					index: index,
					title: ''
				}
				for(var axisIndex=0; axisIndex < 6*24+1; axisIndex++){
					var yaxisStartTime = getYesterDay(6-index).substring(0,10)+ ' 00:00:00',
						offsetTime = addTimes(new Date(yaxisStartTime),axisIndex*(60/chartData.pointerCount));
					chartData.data.push('-');
					chartData.xAxisData.push(formatDate(offsetTime));
				}
				var params = {
					start_time : s_time,
					end_time : e_time,
					timeZone : timeZone,
					time_level: "min",
					device_type:'egw',
					device_code:vm.egwCode
				};
				if(code == 'ueCountChart'){
					chartData.legendNames = ['UE'];
					chartData.yAxisName = '('+'<%=rb.getString("ueGe")%>' + ')';
					chartData.title = 'UE Count';
					chartData.color = ['#69B6FC'];
				}
				if(code == 'upLinkChart'){
					chartData.legendNames = ['UPLink'];
					if(vm.upLinkUnitType == 'MB'){
						chartData.yAxisName = '(MB)';
					}else{
						chartData.yAxisName = '(GB)';
					}
					chartData.title = 'Uplink Traffic';
					chartData.color = ['#90EC97'];
				}
				if(code == 'downLinkChart'){
					chartData.legendNames = ['DownLink'];
					if(vm.downLinkUnitType == 'MB'){
						chartData.yAxisName = '(MB)';
					}else{
						chartData.yAxisName = '(GB)';
					}
					chartData.title = 'Downlink Traffic';
					chartData.color = ['#90EC97'];
				}
				// 图标数据
				axios.post("${ctx}/system/device/getDeviceStatusDataList.action",stringify(params)).then(function(response){
					var data = response.data;

					if(data && data.length>0){
						data.map((item)=>{
							if(chartData.xAxisData.indexOf(item.statistics_time) != -1){
								var ids = chartData.xAxisData.indexOf(item.statistics_time);
								if(code == 'ueCountChart'){
									chartData.data[ids] = item.ueCount;
								}else if(code == 'upLinkChart'){
									if(vm.upLinkUnitType == 'MB'){
										chartData.data[ids] = item.uplink_traffic;
									}else{
										chartData.data[ids] = Number((item.uplink_traffic/1024).toFixed(1));
									}
								}else if(code == 'downLinkChart'){
									if(vm.downLinkUnitType == 'MB'){
										chartData.data[ids] = item.downlink_traffic;
									}else{
										chartData.data[ids] = Number((item.downlink_traffic/1024).toFixed(1));
									}
								}
							}	
						})
					}
					vm.charts[code].setOption(vm.createOption(chartData));
				}) 
				
				
			},
			/**
			* 统计数据刷新  CPU 硬盘 内存
			* @param code{string} 图表类型
			* @param index{number}  时间轴下标
			*/
			getDeviceChartData(code,index){
				var vm = this,
					s_time = getYesterDay(6-index).substring(0,10)+' 00:00:00',
					e_time = getYesterDay(6-index).substring(0,10)+' 23:59:59';
				
				if(index==6) {
					e_time = end_time;
				}else{
					e_time = getYesterDay(6-index-1).substring(0,10)+' 00:00:00';
				}
				var chartData={
					legendNames: [],
					data: [],
					yAxisName: '',
					xAxisData:[],
					color:[],
					pointerCount: 4,
					index: index,
					title: ''
				}
				for(var axisIndex=0; axisIndex < 4*24+1; axisIndex++){
					var yaxisStartTime = getYesterDay(6-index).substring(0,10)+ ' 00:00:00',
						offsetTime = addTimes(new Date(yaxisStartTime),axisIndex*(60/chartData.pointerCount));
					chartData.data.push('-');
					chartData.xAxisData.push(formatDate(offsetTime));
				}
				var params = {
					startTime : s_time,
					endTime : e_time,
					timeZone : timeZone,
					egwCode:vm.egwCode
				};
				if(code == 'cpuChart'){
					chartData.legendNames = ['CPU'];
					chartData.yAxisName = '(%)';
					chartData.title = '<%=rb.getString("CPUShiYongLv")%>';
					chartData.color = ['#69B6FC'];
				}
				if(code == 'memoryChart'){
					chartData.legendNames = ['<%=rb.getString("NeiCun")%>'];
					chartData.yAxisName = '(%)';
					chartData.title = '<%=rb.getString("NeiCunShiYongLv")%>';
					chartData.color = ['#90EC97'];
				}
				if(code == 'diskChart'){
					chartData.legendNames = ['<%=rb.getString("CiPan")%>'];
					chartData.yAxisName = '(%)';
					chartData.title = '<%=rb.getString("CiPanShiYongLv")%>';
					chartData.color = ['#90EC97'];
				}
				// 图表数据
				axios.post("${ctx}/system/device/getEgwStatisticsList.action",stringify(params)).then(function(response){
					var data = response.data;

					if(data && data.length>0){
						data.map((item)=>{
							if(chartData.xAxisData.indexOf(item.startTime) != -1){
								var ids = chartData.xAxisData.indexOf(item.startTime);
								if(code == 'cpuChart'){
									chartData.data[ids] = item.cpu;
								}else if(code == 'memoryChart'){
									chartData.data[ids] = item.memory;
								}else if(code == 'diskChart'){
									chartData.data[ids] = item.disk;
								}
							}	
						})
					}
					vm.charts[code].setOption(vm.createOption(chartData));
					setTimeout(()=>{
						vm.charts[code].resize();
					},200)
				}) 
			},
			// 生成option
			createOption(chartData){
				var vm = this,
					pointerCount = chartData.pointerCount,
					option = {
						baseOption: {
							title: {
								subtext: chartData.title,
								top: -5,
								left:20,
								subtextStyle: {
									color: '#333'
								}
							},
							timeline: {
								axisType: 'category',
								controlPosition: 'none',
								symbolSize:8,
								lineStyle : { color : '#666666',width : 1 },
								itemStyle : {
									normal : { borderColor : '#B0AFBA' },
									emphasis : {
										borderColor : '#1e90ff',
										color : '#1e90ff'
									}
								},
								data: [getYesterDay(6).substring(5).replace("-","."),
									getYesterDay(5).substring(5).replace("-","."),
									getYesterDay(4).substring(5).replace("-","."),
									getYesterDay(3).substring(5).replace("-","."),
									getYesterDay(2).substring(5).replace("-","."),
									getYesterDay(1).substring(5).replace("-","."),
									getYesterDay(0).substring(5).replace("-",".")],
								notMerge:true,
								currentIndex: chartData.index,
								checkpointStyle:{
									color:'#209FFF',
									borderColor:'none'
								}
							},
							tooltip : {
								trigger : 'axis',
							},
							grid:{
								bottom: 70,
								left:10,
								right: 10,
								containLabel:true
							}, 
							color: chartData.color,
							legend: { 
								data: chartData.legendNames,
								top:10
							},
							xAxis : [{
								name : '<%=rb.getString("XiaoShi")%>',
								type : 'category',
								boundaryGap : false,
								axisLine:{
									show : true,
									lineStyle:{ color:"#666666" }
								},
								axisLabel : {
									show:true,
									textStyle:{ color:"#666666" },
									lineStyle:{ color:"#666666" },
									formatter : function(val) {
										var secondTime = val.split(' ')[1];
										clock = secondTime.substring(0,2);
										val = secondTime.substring(0,5);
										if(secondTime.substring(3,5)=='00') return clock;
										return val;
									},
									interval : function(index){
										if(index%pointerCount == 0 && index != pointerCount*24){
											return true;
										}
									},
									rotate : (function(){
											var degree = 0;
											return degree;
									})(),
								}
							}],
							yAxis : [{
								name: chartData.yAxisName,
								minInterval: null,
								type: 'value',
								axisLabel : {
									show:true,
									textStyle:{ color:"#666666" },
									lineStyle:{ color:"#666666" }
								},
								axisLine:{
									lineStyle:{ color:"#666666" }
								},
								splitLine : {
									lineStyle:{ color:"#f1f1f4" }
								}
							}],
							series : [
								{  
									name: chartData.legendNames[0],
									type: 'line',
									symbolSize: 1,
									showAllSymbol: true,
									step: false,
									connectNulls: true,
									itemStyle: {
										normal: {
											areaStyle: {opacity: 0.2}
										}
									},
									areaStyle: {opacity: 0.2},
								}
							]
						},
						options: [
							{  
				              xAxis: [{ data: chartData.xAxisData}],
				              series: [
				                  {data:chartData.data}, 
				              ]  
				            },
							{  
				              xAxis: [{ data: chartData.xAxisData}],
				              series: [
				                  {data:chartData.data}, 
				              ]  
				            },
							{  
				              xAxis: [{ data: chartData.xAxisData}],
				              series: [
				                  {data:chartData.data}, 
				              ]  
				            },
							{  
				              xAxis: [{ data: chartData.xAxisData}],
				              series: [
				                  {data:chartData.data}, 
				              ]  
				            },
							{  
				              xAxis: [{ data: chartData.xAxisData}],
				              series: [
				                  {data:chartData.data}, 
				              ]  
				            },
							{  
				              xAxis: [{ data: chartData.xAxisData}],
				              series: [
				                  {data:chartData.data}, 
				              ]  
				            },
							{  
				              xAxis: [{ data: chartData.xAxisData}],
				              series: [
				                  {data:chartData.data}, 
				              ]  
				            },
						]
					};
				return option;
			},
			// 同步状态更新
			changeSynStatus(){
				var vm = this;

				vm.$confirm('<%=rb.getString("QueDingTongBuSheBei")%>','<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal: false
				}).then(() => {
					var params = {
						egwCode: vm.deviceData.egwCode
					};
					axios.post("${ctx}/egw/monitor/refreshEGWInfo.action",stringify(params)).then(function(response){
						
					})
				}).catch(function(){})
			
			},
			// 重启状态更新
			changeRebootStatus(){
				var vm = this;

				vm.$confirm('<%=rb.getString("QueDingChongQiSheBei")%>','<%=rb.getString("QueRen")%>',{
		    		customClass:'warningConfirm',
		    		confirmButtonText:'<%=rb.getString("QueDing")%>',
		    		cancelButtonText:'<%=rb.getString("QuXiao")%>',
		    		type:'warning',
		    		closeOnClickModal: false
		    	}).then(() => {
		    		var params = {
						egwCode: vm.deviceData.egwCode
					};
					axios.post("${ctx}/egw/reboot/egwReboot.action",stringify(params)).then(function(response){
						
					})
		    	}).catch(function(){})
			},
			//  告警类型改变事件
			alarmTypeChange(type){
				var vm = this;
				if(vm.alarmType == type)return
				vm.alarmType = type;
				vm.alarmQueryParams.alarmType = type;
			},
			/**
			* 日志列表 点击更多操作出现菜单
			* @param row{object}   行数据
			* @param ev{object}   event数据
			*/ 
			optLogClick(row,ev){ // 操作项 
				var vm = this,
					status = row.task_status,
					terminateDis = false,
					deleteDis = false;
				
				vm.rowDataLog = row;
				if(status == 0 || status == 1 || status == 5 || status == 7 || status == 8){
					terminateDis = true;
				}else{
					terminateDis = false;
				}

				//进行中 - 表格操作-删除为置灰状态
				if(status == 1 || status == 5 || status == 7){
					deleteDis = false;
				}else{
					deleteDis = true;
				}

				vm.menusLog= [
					{label:'<%=rb.getString("ZhongZhiRenWu")%>',cls:"el-icon-operation-terminate el-icon",code:'terminate',disable:!terminateDis},
					{label:'<%=rb.getString("XiaZai")%>',cls:"el-icon-operation-download el-icon CODE_EGW hidden" ,code:'download'},
					{label:'<%=rb.getString("ShanChu")%>',cls:"el-icon-operation-delete el-icon CODE_EGW hidden" ,code:'del',disable:!deleteDis},
				];
				
				vm.$nextTick(function(){
					document.body.click();
					vm.$refs.menusLog.show(ev);
				});
			},
			clickLogMenu(ev){
				var vm = this,
					codes = {
						terminate:vm.terminateCollectTask,
						download:vm.downlodLogFile,
						del:vm.delCollectTask
					};
				if(codes[ev.code]){
					codes[ev.code](vm.rowDataLog)
				}
			},
			// 终止日志收集任务
			terminateCollectTask(row){
				var vm = this,
					status = row.task_status,
					urls = '${ctx}/cell/collect/goTerminateImmediateCollectLogFile.action',
					params = {
						taskIds:row.task_id,
						device_code:row.device_code,
						execute_type:row.execute_type 
					};
				if (status != 0 && status != 5 && status != 1 && status != 7 && status != 8) {
					showMsg('prompt_msg','<%=rb.getString("MeiYouKeTingZhiShouJiDeSheBei")%>');
					return;
				}
				axios.post(urls,stringify(params)).then(function(response){
					var data = response.data;
					if(data["success"]){
						vm.$message.success( '<%=rb.getString("ChengGong")%>');
						vm.$refs.logTable.refresh();//表格刷新
					}else{
						vm.$message.error(data.message) //错误提示信息
					}
				}) 
			},
			// 下载日志文件
			downlodLogFile(row){
				var vm = this,
					urls = '${ctx}/cell/collect/getDownloadFileNumber.action',
					params = {
						taskIds: row.task_id
					};

				axios.post(urls,stringify(params)).then(function(response){
					var data = response.data;
					if(data.length>0){
						 exportByForm("${ctx}/cell/collect/doDownloadImmediateCollectLogFile.action",{
							timeZone: timeZone,
							taskIds: row.task_id,
						});
					}else{
						vm.$message.error('<%=rb.getString("WenJianBuCunZai")%>')
					}
				}) 
			},
			// 删除日志任务文件
			delCollectTask(row){
				var vm = this,
					urls = '${ctx}/cell/collect/doClearImmediateCollectLogFile.action',
					params = {
						taskIds: row.task_id,
						fileName:''
					};
				if(row.file_name){
					params.fileName = row.file_name;
				}
				vm.$confirm('<%=rb.getString("QueRenShanChuRenWuJiLuHeWenJian")%>','<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
				}).then(function(){
					axios.post(urls,stringify(params)).then(function(response){
						var data = response.data;
						if(data["success"]){
							vm.$message.success('<%=rb.getString("ChengGong")%>')
							vm.$refs.logTable.refresh();//表格刷新
						}else{
							vm.$message.error(data.message) //错误提示信息 
						}
					}) 
				}).catch(function(){
					
				})
			},
			/**
			* 告警列表 点击更多操作出现菜单
			* @param row{object}   行数据
			* @param ev{object}   event数据
			*/ 
			optAlarmClick(row,ev){ // 操作项 
				var vm = this ,
					showClear = true,
					confirmDis = true;
				
				vm.rowDataAlarm = row;
				if(row.alarm_type == 'active' ){
					showClear = true;
				}else{
					showClear = false;
				}
				if(row.deal_state == '0' || row.deal_state == '2'){
					confirmDis = false;
				}
				vm.menusAlarm= [
					{label:'<%=rb.getString("XiangXi")%>',cls:"el-icon-operation-info el-icon",code:'detail'},
					{label:'<%=rb.getString("QueRenGaoJing")%>',cls:"el-icon-operation-confirm el-icon CODE_EGW hidden" ,code:'confirm'},
					{label:'<%=rb.getString("FanQueRenGaoJing")%>',cls:"el-icon-operation-unconfirm el-icon CODE_EGW hidden" ,code:'unConfirm',disable:!confirmDis},
					{label:'<%=rb.getString("QingChuGaoJing")%>',cls:"el-icon-operation-clear el-icon CODE_EGW hidden",code:'clear',show:showClear},
					{label:'<%=rb.getString("ShanChuGaoJing")%>',cls:"el-icon-operation-delete el-icon CODE_EGW hidden",code:'del',show:!showClear}
				];
				
				vm.$nextTick(function(){
					document.body.click();
					vm.$refs.menusAlarm.show(ev);
				});
			},
			//点击页面其他地方菜单收起
			handerClose(){ 
				this.$refs.menusAlarm.hide();
				this.$refs.menusLog.hide();
			},
			/**
			* 告警菜单 点击事件
			* @param ev{object}   行数据
			*/ 
			clickAlarmMenu(ev){ //单点击方法 --操作  
				var vm = this,
					codes = {
						detail:vm.detailAlarmInfo,
						confirm:vm.openConfirmDialog,
						unConfirm:vm.unconfirmAlarm,
						clear:vm.clearAlarm,
						del:vm.deleteAlarm
					};
				if(codes[ev.code]){
					codes[ev.code](vm.rowDataAlarm)
				}
			},
			// 告警详情
			detailAlarmInfo(){
				var vm = this;
				vm.alarmInfoDialogUrl = "${ctx}/cell/fault/goAlarmDetail.action?timeZone="+ timeZone;
				var delDom = document.querySelector('#alarmDetailPage');
				if(delDom){
					delDom.remove();
				}
				vm.showAlarmInfo = true;
			},
			// 告警确认
			openConfirmDialog(row){
				var vm = this,
					status = row.deal_state;
				if(status == '1' || status == '3'){
					vm.confirmFlag = true;
				}else{
					vm.confirmFlag = false;
				}
				vm.showConfirmAlarmDialog = true;
				axios.post('${ctx}/cell/fault/queryAlarmDetail.action',stringify({
					alarm_id : vm.rowDataAlarm.alarm_id,
					type:vm.rowDataAlarm.alarm_type,
					timeZone:timeZone
				})).then(function(response){
					var data = response.data;
					vm.confirmForm.confirmUser = data.DEAL_USER;
					vm.confirmForm.confirmTime = data.DEAL_TIME;
					vm.confirmForm.description = data.DEAL_MEMO;
				}) 
			},
			// 告警确认提交
			confirmAlarmSubmit(){
				var vm = this,
					url = '${ctx}/cell/fault/confirmAlarm.action',   //确认告警 
					msg = '<%=rb.getString("QueRenGaoJingTiShi")%>';
				
				axios.post(url,stringify({
					alarm_id : vm.rowDataAlarm.alarm_id,
					small_cell_code: vm.rowDataAlarm.small_cell_code,
					type : vm.rowDataAlarm.alarm_type,
					text : vm.confirmForm.description
				})).then(function(response){
					var data = response.data;
					if(data["success"]){
						vm.confirmAlarmDialogClose();
						vm.$message.success( '<%=rb.getString("ChengGong")%>');
						vm.$refs.alarmTable.refresh();//表格刷新
					}else{
						vm.confirmAlarmDialogClose();
						vm.$message.error(msg + '<%=rb.getString("TiShiShiBai")%>') //错误提示信息
					}
				}) 
			},
			// 告警反确认
			unconfirmAlarm(row){
				var vm = this;
				vm.$confirm('<%=rb.getString("QueDingQuXiaoGaoJingQueRen")%>','<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
				}).then(function(){
					axios.post('${ctx}/cell/fault/cancelConfirmAlarm.action',stringify({
						alarm_id : row.alarm_id,
						type:row.alarm_type
					})).then(function(response){
						var data = response.data;
						if(data["success"]){
							vm.$message.success('<%=rb.getString("ChengGong")%>');
							vm.$refs.alarmTable.refresh();//表格刷新
						}else{
							vm.$message.error('<%=rb.getString("FanQueRenGaoJingTiShi")%><%=rb.getString("TiShiShiBai")%>') //错误提示信息
						}
					})
				}).catch(function(){
					
				})
			},
			// 清除告警
			clearAlarm(){
				var vm = this;
				vm.showClearAlarmDialog = true;
			},
			// 清除告警提交事件
			clearAlarmSubmit(){
				var vm = this,
					url = '${ctx}/cell/fault/clearAlarm.action',   //确认告警 
					msg = '<%=rb.getString("QingChuGaoJingTiShi")%>';
				
				axios.post(url,stringify({
					alarm_id : vm.rowDataAlarm.alarm_id,
					small_cell_code: vm.rowDataAlarm.small_cell_code,
					type : vm.rowDataAlarm.alarm_type,
					text : vm.clearForm.description
				})).then(function(response){
					var data = response.data;
					if(data["success"]){
						vm.clearAlarmDialogClose();
						vm.$message.success( '<%=rb.getString("ChengGong")%>');
						vm.$refs.alarmTable.refresh();//表格刷新
					}else{
						vm.clearAlarmDialogClose();
						vm.$message.error(msg + '<%=rb.getString("TiShiShiBai")%>') //错误提示信息
					}
				}) 
			},
			// 删除告警
			deleteAlarm(row){
				var vm = this;

				vm.$confirm('<%=rb.getString("QueRenShanChuGaoJing")%>','<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
				}).then(function(){
					axios.post('${ctx}/cell/fault/clearHistoryAlarm.action',stringify({
						alarm_id : row.alarm_id,
						small_cell_code:row.small_cell_code,
					})).then(function(response){
						var data = response.data;
						if(data["success"]){
							vm.$message.success('<%=rb.getString("ChengGong")%>')
							vm.$refs.alarmTable.refresh();//表格刷新
						}else{
							vm.$message.error('<%=rb.getString("ShanChuGaoJingShiBai")%>') //错误提示信息 
						}
					}) 
				}).catch(function(){
					
				})
			},
			// 告警详情弹窗打开成功事件
			openAlarmInfoDialogSuc(){
				var vm = this;
				eventBus.$emit('detail-info',vm.rowDataAlarm.alarm_type,vm.rowDataAlarm.alarm_id);
			},
			// 告警详情弹窗关闭
			closeAlarmInfoDialog(){
				var vm = this;
				vm.showAlarmInfo = false;
				vm.alarmInfoDialogUrl = '';
			},
			// 清除弹窗关闭事件
			clearAlarmDialogClose(){
				var vm = this;
				vm.showClearAlarmDialog = false;
				vm.clearForm.description = '';
			},
			// 确认弹窗关闭事件
			confirmAlarmDialogClose(){
				var vm = this,
					params = {
						confirmUser:'',
						confirmTime:'',
						description:''
					};
				vm.showConfirmAlarmDialog = false;
				Object.assign(vm.confirmForm,params);
			},
			// 日志收集事件
			collectLogsClick(){
				var vm = this,
					params = {
						timeZone: timeZone,
						serial_number: vm.deviceData.egwSn,
						isReboot: 'false',
						device_code: vm.deviceData.egwCode,
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
						vm.$refs.logTable.refresh();//表格刷新
					}else if(data.responseCode == "901"){
						vm.$message.error(data.message);
					}else if(data.responseCode == "401"){//存在未完成的任务时，再次创建任务失败，给出提示
						vm.$message.error(data["message"]);
					}else{
						vm.$message.error('<%=rb.getString("ShouJiShiBai")%>');
					}
				})
			},
			// 上行流量单位切换
			upLinkUnitTypeChange(val){
				var vm = this;
				if(vm.upLinkUnitType == val)return
				vm.upLinkUnitType = val;
				vm.proccessChartData('upLinkChart',vm.currenUpLinkIndex);
			},
			// 下行流量单位切换
			downLinkUnitTypeChange(val){
				var vm = this;
				if(vm.downLinkUnitType == val)return
				vm.downLinkUnitType = val;
				vm.proccessChartData('downLinkChart',vm.currenDownLinkIndex);
			}
			
		},
		created(){},
		mounted(){
			var vm = this;
			eventBus.$off('info-init').$on('info-init',this.init);
			clearInterval(egwStatusTimer);
			egwStatusTimer = undefined;
			egwStatusTimer = setInterval(function(){
				vm.checkStatus();
			},6000)
		}
		
	});
</script>