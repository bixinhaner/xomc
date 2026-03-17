<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>

<style type="text/css">
	#cpeFactoryResetTaskContent .tow-line-split {
		height: 100%;
		display: flex;
		flex-direction: column;
	}
	#cpeFactoryResetTaskContent .line-flex-item {
		flex: auto;
		overflow: auto;
	}	
	#cpeFactoryResetTaskContent .result-info {
		display: flex;
		align-items: center;
		height: 16px;
		line-height: 16px;
		margin-right: 10px;
		border-right: 1px solid #DFE2EE;
		color: #4D84FF;
		background: #FFFFFF;
		font-size: 14px;
	}
	#cpeFactoryResetTaskContent .result-info i {
		margin-right: 5px;
	}
	#cpeFactoryResetTaskContent .result-count {
		display: flex;
		align-items: center;
		justify-content: center;
		height: 100%;
		min-width: 30px;
		text-align: center;
		color: rgba(0, 0, 0, 0.8);	
		background: #FFFFFF;
	}

	#cpeFactoryResetTaskContent .failure-color .el-icon-circle-close:before{
		content:'\e6fb';
	}
	#cpeFactoryResetTaskContent .success-color, 
	#cpeFactoryResetTaskContent .el-icon.success-color::before,
	#cpeFactoryResetTaskContent .success-color .el-icon-operation-defaultBeta::before,
	#cpeFactoryResetTaskContent .status-info .el-icon-circle-success::before {
		color: #67d972;
	}
	#cpeFactoryResetTaskContent .failure-color, 
	#cpeFactoryResetTaskContent .el-icon.failure-color::before,
	#cpeFactoryResetTaskContent .failure-color .el-icon-circle-close::before,
	#cpeFactoryResetTaskContent .status-info .el-icon-circle-close::before {
		color: #e88282;
	}
	#cpeFactoryResetTaskContent .success-color .el-icon-operation-defaultBeta,
	#cpeFactoryResetTaskContent .failure-color .el-icon-circle-close {
		font-size: 16px;
	}
	#cpeFactoryResetTaskContent .query-form-cls {
		display: flex;
	}
	#cpeFactoryResetTaskContent .query-form-cls .el-form-item {
		display: inline-block;
		margin-right: 40px;
	}
	#cpeFactoryResetTaskContent .slide-position-top > .el-card > .el-card__header .clearfix span .el-icon-close{
		display:none !important;
	}
	#cpeFactoryResetTaskContent .operations{
		position: absolute;
		top: -3px;
		right: 10px;
		z-index: 100;
	}
	#cpeFactoryResetTaskContent .searchWarp{
		display: flex;
		align-items: center;
	}
	#cpeFactoryResetTaskContent .cpeListTitle{
		padding-left: 10px;
	}
	#cpeFactoryResetTaskContent .taskListWarp{
		height: 48%;
	}
	#cpeFactoryResetTaskContent .taskListNumberWarp{
		display: flex;
		position: absolute;
		right: 0px;
		top: 18px;
	}
	#cpeFactoryResetTaskContent .deviceListSearchWarp{
		display: flex;
		justify-content: space-between;
	}
	#cpeFactoryResetTaskContent .h100{
		height: 100%;
	}

	#cpeFactoryResetTaskContent input::-webkit-input-placeholder,
	#cpeFactoryResetTaskContent input::-moz-input-placeholder,
	#cpeFactoryResetTaskContent input::-ms-input-placeholder{
		color:#CCCCCC !important;
	}
	
	#cpeFactoryResetTaskContent .commonTabsTop .el-tabs__header{
		border: 1px solid #D5DCEC;
		border-radius: 8px 8px 0 0;
	}
</style>

<!-- cPE恢复出厂 -->
<div class="panelDefault" id='cpeFactoryResetTaskContent' style='border: 0; background: unset;'>
	<!-- 操作按钮 -->
	<div class="operations">
		<div v-if="hasConfigRole" class="placeholder-bt circleIcon" placeholder="<%=rb.getString("XinZeng")%>">
			<span class="el-icon el-icon-circle-add" @click="toAddTask"></span>
		</div>
	</div>
	<div class="tow-line-split">
		
		<div class="line-flex-item headerTableWarp">
			<!--batch cpe  表格 -->
			<el-tabs v-model='activeName' class="h100 commonTabsTop" @tab-click='tabClick'>
				<el-tab-pane label='<%=rb.getString("CPEPiLiangPeiZhi") %>' name='batch'>
					<div style="height:50%; " class='commonWarp commonTableBorder'>
						<el-ctable ref="batchCpeList" id="cpeListTable" 
							:url="cpeListUrl"
							:query-params="batchCpeListParams"
							@selection-change="cpeListSelectChange">
							<template slot="toolbar" style="position:relative;">
								<div class="commonFlex commonQuery">
									<p class="subTitle"><%=rb.getString("SuoPinCPELieBiao")%></p>
									<el-query type="normal" @query="batchCpeListQuery" placeholder="<%=rb.getString("CPEXuLieHao")%> / <%=rb.getString("CPEName")%> / PCI"></el-query>
								</div>
								<el-radio-group size="mini" v-model='cpe_model' class="commonRadioButton" @change="cpeModelClick" style='width:155px;position:absolute;right:20px;top:10px;'>
									<el-radio-button label="4G">4G CPE</el-radio-button>
									<el-radio-button label="5G">5G CPE</el-radio-button>
								</el-radio-group>
							</template>
							<el-table-column v-if="hasConfigRole" type="selection" :resever-selection="true"></el-table-column>
							<el-table-column prop="connection_status" width="50">
								<template slot-scope="scope">
									<div :class="{
										'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
										'':scope.row.have_connected==2,
										'conn_exc':scope.row.connection_status=='Exception',
										'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
								</template>
							</el-table-column>
							<el-table-column label="<%=rb.getString("CPEXuLieHao")%>" prop="serial_number"></el-table-column>
							<el-table-column label="<%=rb.getString("CPEName")%>" prop="host_name"></el-table-column>
							<el-table-column label="IMSI" prop="imsi"></el-table-column>
							<el-table-column label="<%=rb.getString("ChanPinXingHao")%>" prop="model_name"></el-table-column>
							<el-table-column label="<%=rb.getString("MACDiZhi")%>" prop="mac_address"></el-table-column>
							<el-table-column label="<%=rb.getString("IPDiZhi")%>" prop="ipaddress"></el-table-column>
							<el-table-column label="PCI" prop="pci"></el-table-column>
							<el-table-column label="<%=rb.getString("SheBeiZu")%>" prop="group_name"></el-table-column>
						</el-ctable>
					</div>
					<div style="height: calc(50% - 10px); margin-top: 10px;" class="line-flex-items">
						<el-tabs style="height: 100%;" class="  ">
							<!-- 批量配置  task list,device list -->
							<el-tab-pane label="<%=rb.getString("RenWuLieBiao")%>" >
								<el-ctable ref="batchTaskList" :time="6" id="batchTaskListTable" style='border: 1px solid #D5DCEC;border-top: 0; border-radius: 0 0 8px 8px;overflow: hidden;box-sizing: border-box;'
									:url="batchTaskListUrl" 
									:query-params="batchTaskListParams"
									@load-success="batchTaskListLoadSuccess">
									<template slot="toolbar">
										<div class='toolbarHeadBtnBoxCls commonQuery' style="height:40px; padding: 0 0 10px 0;">
											<el-query type="normal" @query="batchTaskListQuery" placeholder="<%=rb.getString("RenWuMingCheng")%>"></el-query>
											<el-date-picker style='margin-left: 10px;' 
												v-model="dateValue"
												type="datetimerange"
												value-format="yyyy-MM-dd HH:mm:ss"
												range-separator="——"  
												@change="dateChange"
												start-placeholder='<%=rb.getString("KaiShiShiJian")%>' 
												end-placeholder='<%=rb.getString("JieShuShiJian")%>'>
											</el-date-picker>
										</div>
								
										<div class="taskListNumberWarp">
											<div class="result-info">
												<i class="el-icon el-icon-status-waiting1"></i> <%=rb.getString("DengDai")%>
												<span class="result-count">{{bactchTaskStatistic.WaitingNum}}</span>
											</div>
											<div class="result-info">
												<i class="el-icon el-icon-status-inProgress"></i> <%=rb.getString("JinXingZhong")%>
												<span class="result-count">{{bactchTaskStatistic.ProcessingNum}}</span>
											</div>
											<div class="result-info">
												<i class="el-icon el-icon-status-terminate"></i> <%=rb.getString("ZanTing")%>
												<span class="result-count">{{bactchTaskStatistic.SuspendedNum}}</span>
											</div>
											<div class="result-info" style='border: 0;'>
												<i class="el-icon el-icon-status-success"></i> <%=rb.getString("JieShu")%>
												<span class="result-count">{{bactchTaskStatistic.EndNum}}</span>
											</div>
										</div>
									</template>
									<el-table-column width="40">
										<template slot-scope="scope">
											<div class="el-icon el-icon-operation-more" @click="optClick(scope.row, event)"  v-clickoutside="handerClose"></div>
										</template>
									</el-table-column>
									
									<el-table-column label="<%=rb.getString("RenWuMingCheng")%>" prop="TASK_NAME" min-width="260"></el-table-column>
									<el-table-column label="<%=rb.getString("CaoZuoRen")%>" prop="CREATE_USER" min-width="100"></el-table-column>
									<el-table-column label="<%=rb.getString("CaoZuoShiJian")%>" prop="CREATE_TIME" min-width="150"></el-table-column>
									<el-table-column label="<%=rb.getString("ZhuangTai")%>" prop="TASK_STATUS" min-width="150">
										<template slot-scope="scope">
											<div v-html="taskTableStatus(scope.row.TASK_STATUS)"></div>
										</template>
									</el-table-column>
									<el-table-column label="<%=rb.getString("JinDu")%>" prop="TASK_PROGRESS" min-width="130"></el-table-column>
									<el-table-column label="<%=rb.getString("JieGuo")%>" prop="TASK_RESULT" :formatter="resultFmt" min-width="150"></el-table-column>
									<el-table-column label="<%=rb.getString("KaiShiShiJian")%>" prop="START_TIME" min-width="150"></el-table-column>
									<el-table-column label="<%=rb.getString("JieShuShiJian")%>" prop="END_TIME" min-width="150"></el-table-column>
								</el-ctable>					
								<el-cmenu ref="batchMenu" :data="menus" @click="clickMenu"></el-cmenu>					
							</el-tab-pane>
						
							<el-tab-pane label="<%=rb.getString("BackupRestoreSheBeiLieBiao")%>">
								<el-ctable ref="batchDeviceList" :time="6"  id="batchDeviceListTable" style='border: 1px solid #D5DCEC;border-top: 0; border-radius: 0 0 8px 8px;overflow: hidden;box-sizing: border-box;'
									:url="batchDeviceListUrl"
									:query-params="batchDeviceListParams"
									@load-success="batchDeviceListLoadSuccess">
									<template slot="toolbar">
										<div class="deviceListSearchWarp commonFlex commonQuery">
											<el-query @query="batchDeviceListQuery" placeholder="<%=rb.getString("CPEXuLieHao")%> / <%=rb.getString("CPEName")%> /<%=rb.getString("RenWuMingCheng")%>" type="normal"></el-query>
											
											<div style="display: flex; margin-right: 40px; margin-top: 5px;">
												<div class="result-info success-color">
													<i class="el-icon el-icon-operation-defaultBeta"></i> <%=rb.getString("ChengGong")%>
													<span class="result-count">{{batchTaskDeviceStatistic.SucNum}}</span>
												</div>
												<div class="result-info failure-color" style='border: 0'>
													<i class="el-icon el-icon-circle-close"></i> <%=rb.getString("ShiBai")%>
													<span class="result-count">{{batchTaskDeviceStatistic.FailNum}}</span>
												</div>
												<div class="newIconBoxCls-bt" style="right:15px;top:10px;" @click="exportDevice" tip="<%=rb.getString("DaoChu")%>">
													<span class="el-icon el-icon-circle-export"></span>
												</div>
											</div>
										</div>
									</template>
									<el-table-column label="<%=rb.getString("CPEXuLieHao")%>" prop="SERIAL_NUMBER" min-width="160"></el-table-column>
									<el-table-column label="IMSI" prop="IMSI" min-width="140"></el-table-column>
									<el-table-column label="<%=rb.getString("CPEName")%>" prop="HOST_NAME" min-width="160"></el-table-column>
									<el-table-column label="<%=rb.getString("RenWuMingCheng")%>" prop="TASK_NAME" min-width="220"></el-table-column>
									<el-table-column label="<%=rb.getString("ZhuangTai")%>" prop="PROGRESS_STATUS" min-width="120">
										<template slot-scope="scope">
											<div v-html="taskTableStatus(scope.row.PROGRESS_STATUS)"></div>
										</template>
									</el-table-column>
									<el-table-column label="<%=rb.getString("JieGuo")%>" prop="PROGRESS_RESULT" :formatter="resultFmt" min-width="110"></el-table-column>
									<el-table-column label="<%=rb.getString("ShiBaiYuanYin")%>" prop="FAILURE_REASON" min-width="130"></el-table-column>
									<el-table-column label="<%=rb.getString("KaiShiShiJian")%>" prop="START_TIME" min-width="130"></el-table-column>
									<el-table-column label="<%=rb.getString("JieShuShiJian")%>" prop="END_TIME" min-width="130"></el-table-column>
								</el-ctable>
							</el-tab-pane>				
						</el-tabs>
					</div>								
				</el-tab-pane>				
				
				<!-- 恢复出厂 -->
				<el-tab-pane name='restore' label="<%=rb.getString("CPEHuiFuChuChang")%>" v-if="false">
					<div style="height:50%;">
						<el-ctable ref="cpeList" id="cpeListTable" 
						:url="cpeRestoreListUrl"
						:query-params="cpeListParams"
						@selection-change="cpeListSelectChange">
						<template slot="toolbar">
							<div class="searchWarp">
								<h3 class="cpeListTitle"><%=rb.getString("SuoPinCPELieBiao")%></h3>
								<el-query type="normal" @query="cpeListQuery" placeholder="<%=rb.getString("CPEXuLieHao")%> / <%=rb.getString("CPEName")%>"></el-query>
							</div>
						</template>
						<el-table-column type="selection" :resever-selection="true"></el-table-column>
						<el-table-column prop="connection_status" width="50">
							<template slot-scope="scope">
								<div :class="{
									'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
									'':scope.row.have_connected==2,
									'conn_exc':scope.row.connection_status=='Exception',
									'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
							</template>
						</el-table-column>
						<el-table-column label="<%=rb.getString("CPEXuLieHao")%>" prop="serial_number"></el-table-column>
						<el-table-column label="<%=rb.getString("CPEName")%>" prop="host_name"></el-table-column>
						<el-table-column label="IMSI" prop="imsi"></el-table-column>
						<el-table-column label="<%=rb.getString("ChanPinXingHao")%>" prop="model_name"></el-table-column>
						<el-table-column label="<%=rb.getString("MACDiZhi")%>" prop="mac_address"></el-table-column>
						<el-table-column label="<%=rb.getString("IPDiZhi")%>" prop="ipaddress"></el-table-column>
						<el-table-column label="<%=rb.getString("SheBeiZu")%>" prop="group_name"></el-table-column>
					</el-ctable>
					</div>
					<div style="padding: 5px;"></div>
					<div style="height:48%;" class="line-flex-item taskListWarp">
					
						<el-tabs style="height: 99%;" class="active-line-top">
							<!-- 恢复出厂  task list,device list -->
							<el-tab-pane label="<%=rb.getString("RenWuLieBiao")%>" >
								<el-ctable ref="taskList" :time="6" id="taskListTable"
									:url="taskListUrl"
									:query-params="taskListParams"
									@load-success="taskListLoadSuccess">
									<template slot="toolbar">
											<el-query placeholder="<%=rb.getString("RenWuMingCheng")%>"
												@query="taskListQuery"
												@advance-query="taskListQueryAdvance"
												@reset="taskListReset">
												<template slot="form">
													<el-form ref="taskListForm" label-position="top" class="query-form-cls">
														<el-form-item label="<%=rb.getString("RenWuMingCheng")%>">
															<el-input v-model="taskListQueryForm.searchText"></el-input>
														</el-form-item>
														<el-form-item label="<%=rb.getString("ShiJian")%>">
															<el-date-picker v-model="taskListQueryForm.time" type="datetimerange" value-format="yyyy-MM-dd HH:mm:ss" size="mini"></el-date-picker>
														</el-form-item>
													</el-form>
												</template>
											</el-query>
											<div class="taskListNumberWarp">
												<div class="result-info">
													<i class="el-icon el-icon-status-waiting1"></i> <%=rb.getString("DengDai")%>
													<span class="result-count">{{taskStatistic.WaitingNum}}</span>
												</div>
												<div class="result-info">
													<i class="el-icon el-icon-status-inProgress"></i> <%=rb.getString("JinXingZhong")%>
													<span class="result-count">{{taskStatistic.ProcessingNum}}</span>
												</div>
												<div class="result-info">
													<i class="el-icon el-icon-status-terminate"></i> <%=rb.getString("ZanTing")%>
													<span class="result-count">{{taskStatistic.SuspendedNum}}</span>
												</div>
												<div class="result-info">
													<i class="el-icon el-icon-status-success"></i> <%=rb.getString("JieShu")%>
													<span class="result-count">{{taskStatistic.EndNum}}</span>
												</div>
											</div>
									</template>
									<el-table-column width="40">
										<template slot-scope="scope">
											<div class="el-icon el-icon-operation-more" @click="optClick(scope.row, event)"  v-clickoutside="handerClose"></div>
										</template>
									</el-table-column>
									<el-table-column label="<%=rb.getString("RenWuMingCheng")%>" prop="TASK_NAME" min-width="260"></el-table-column>
									<el-table-column label="<%=rb.getString("ShiFouBaoLiuPeiZhi")%>" prop="KEEP_CONFIG" :formatter="keepCongigFmt" min-width="190"></el-table-column>
									<el-table-column label="<%=rb.getString("CaoZuoRen")%>" prop="CREATE_USER" min-width="100"></el-table-column>
									<el-table-column label="<%=rb.getString("CaoZuoShiJian")%>" prop="CREATE_TIME" min-width="150"></el-table-column>
									<el-table-column label="<%=rb.getString("ZhuangTai")%>" prop="TASK_STATUS" min-width="150">
										<template slot-scope="scope">
											<div v-html="taskTableStatus(scope.row.TASK_STATUS)"></div>
										</template>
									</el-table-column>
									<el-table-column label="<%=rb.getString("JinDu")%>" prop="TASK_PROGRESS" min-width="130"></el-table-column>
									<el-table-column label="<%=rb.getString("JieGuo")%>" prop="TASK_RESULT" :formatter="resultFmt" min-width="150"></el-table-column>
									<el-table-column label="<%=rb.getString("KaiShiShiJian")%>" prop="START_TIME" min-width="150"></el-table-column>
									<el-table-column label="<%=rb.getString("JieShuShiJian")%>" prop="END_TIME" min-width="150"></el-table-column>
								</el-ctable>					
								<el-cmenu ref="menu" :data="menus" @click="clickMenu"></el-cmenu>					
							</el-tab-pane>
							
							<el-tab-pane label="<%=rb.getString("BackupRestoreSheBeiLieBiao")%>">
								<el-ctable ref="deviceList" :time="6"  id="deviceListTable" 
									:url="deviceListUrl"
									:query-params="deviceListParams"
									@load-success="deviceListLoadSuccess">
									<template slot="toolbar">
										<div class="deviceListSearchWarp">
											<el-query @query="deviceListQuery" placeholder="<%=rb.getString("CPEXuLieHao")%> / <%=rb.getString("CPEName")%> / <%=rb.getString("RenWuMingCheng")%>" type="normal"></el-query>
											
											<div style="display: flex; margin-right: 40px; margin-top: 5px;">
												<div class="result-info success-color">
													<i class="el-icon el-icon-operation-defaultBeta"></i> <%=rb.getString("ChengGong")%>
													<span class="result-count">{{taskDeviceStatistic.SucNum}}</span>
												</div>
												<div class="result-info failure-color" style='border: 0;'>
													<i class="el-icon el-icon-circle-close"></i> <%=rb.getString("ShiBai")%>
													<span class="result-count">{{taskDeviceStatistic.FailNum}}</span>
												</div>
												<div class="newIconBoxCls-bt" style="right:15px;top:10px;" @click="exportDevice" tip="<%=rb.getString("DaoChu")%>">
													<span class="el-icon el-icon-circle-export"></span>
												</div>
											</div>
										</div>
									</template>
									<el-table-column label="<%=rb.getString("CPEXuLieHao")%>" prop="SERIAL_NUMBER" min-width="160"></el-table-column>
									<el-table-column label="<%=rb.getString("CPEName")%>" prop="HOST_NAME" min-width="160"></el-table-column>
									<el-table-column label="<%=rb.getString("RenWuMingCheng")%>" prop="TASK_NAME" min-width="220"></el-table-column>
									<el-table-column label="<%=rb.getString("ShiFouBaoLiuPeiZhi")%>" prop="KEEP_CONFIG" :formatter="keepCongigFmt" min-width="190"></el-table-column>
									<el-table-column label="<%=rb.getString("ZhuangTai")%>" prop="PROGRESS_STATUS" min-width="120">
										<template slot-scope="scope">
											<div v-html="taskTableStatus(scope.row.PROGRESS_STATUS)"></div>
										</template>
									</el-table-column>
									<el-table-column label="<%=rb.getString("JieGuo")%>" prop="PROGRESS_RESULT" :formatter="resultFmt" min-width="110"></el-table-column>
									<el-table-column label="<%=rb.getString("ShiBaiYuanYin")%>" prop="FAILURE_REASON" min-width="130"></el-table-column>
									<el-table-column label="<%=rb.getString("KaiShiShiJian")%>" prop="START_TIME" min-width="130"></el-table-column>
									<el-table-column label="<%=rb.getString("JieShuShiJian")%>" prop="END_TIME" min-width="130"></el-table-column>
								</el-ctable>
							</el-tab-pane>
						</el-tabs>
					</div>
				</el-tab-pane>
			</el-tabs>	
		</div>
		
	</div>

	<!-- 新建、查看、修改任务浮层 -->
	<el-slide class='ignore-border commonBorderSlide' ref="slide" 
		:url="slideUrl" 
		:modal='slideModal' 
		:title="slideTitle" 
		:footer="slideFooter" 
		:header='slideHeader' 
		:position="slidePosition"
	 	:height="slideHeight" 
	 	:width="slideWidth" >	 	
	 </el-slide>
</div>
<script type="text/javascript">
	var cpeFactoryResetTaskVue = new Vue({
		el:'#cpeFactoryResetTaskContent',
		data() {
			return {
				activeName:"batch",
				///////
				menus: [],			
				//cpe list
				cpeListUrl: '${ctx}/cell/cpeinfos/queryCpeInfosListForCpe.action?forSelect=3',
				cpeRestoreListUrl: '',
				batchCpeListParams: {
					timeZone: timeZone,
					isShowSlave: false,
					serial_number: '',
					host_name: '',
					search_text: '',
					//以下参数无用
					group_id: '',
					productValue: '',
					cpe_model:'4G'
				},
				// 恢复出厂 cpe list
				cpeListParams: {
					timeZone: timeZone,
					isShowSlave: false,
					serial_number: '',
					host_name: '',
					search_text: '',
					//以下参数无用
					group_id: '',
					productValue: ''
				},
				selection: [],	
				batchTaskListUrl:'${ctx}/cpe/batchconfig/getTaskListInfo.action',
				//task list 
				taskListUrl: '${ctx}/cpe/factoryReset/getTaskListInfo.action',
				taskListQueryForm: {
					searchText: '',
					time: ''
				},
				batchTaskListQueryForm: {
					searchText: '',
					time: ''
				},
				taskListParams: {
					searchText: '',
					timeZone: timeZone,
					startTime: '',
					endTime: '',
					likeFields: 'task_name'
				},
				batchTaskListParams: {
					searchText: '',
					timeZone: timeZone,
					startTime: '',
					endTime: '',
					likeFields: 'task_name'
				},
				bactchTaskStatistic: {
					WaitingNum: '0',
					ProcessingNum: '0',
					SuspendedNum: '0',
					EndNum: '0'
				},
				taskStatistic: {
					WaitingNum: '0',
					ProcessingNum: '0',
					SuspendedNum: '0',
					EndNum: '0'
				},
				//device list
				batchDeviceListUrl: '${ctx}/cpe/batchconfig/getDeviceListInfo.action',
				deviceListUrl: '${ctx}/cpe/factoryReset/getDeviceListInfo.action',
				batchDeviceListParams: {
					searchText: '',
					timeZone: timeZone,
				},
				deviceListParams: {
					searchText: '',
					timeZone: timeZone,
				},
				batchTaskDeviceStatistic: {
					SucNum: '0',
					FailNum: '0'
				},
				taskDeviceStatistic: {
					SucNum: '0',
					FailNum: '0'
				},
				slideUrl:'',
				slideTitle:'',
				slideFooter:'',
				slideHeader:'',
				slidePosition:'',
				slideHeight:'',
				slideWidth:'',
				slideModal:'',
				
				dateValue:[],
				cpe_model:'4G'
			};
		},
		computed: {
			hasConfigRole() {
				return writableMap.CODE_CPE_CONFIG == true;
			}
		},
		methods: {
			init(){
				var vm = this;
			},
			tabClick(){
				var vm = this;
				this.$nextTick(function(){
					document.body.click();
				});
				if(vm.activeName == 'batch'){		

				}else{
					vm.batchTaskListParams.searchText = '';
					vm.cpeRestoreListUrl = '${ctx}/cell/cpeinfos/queryCpeInfosListForCpe.action?forSelect=3';
				}
			},
			
			//cpe list 选择数据
			cpeListSelectChange(selection) {
				this.selection = selection;
			},	
			batchTaskListLoadSuccess(data) {
				var vm = this;				
				if(data && data.properties) {
					Object.assign(vm.bactchTaskStatistic, data.properties);
				}
			},
			//task list 表格数据加载成功回调
			taskListLoadSuccess(data) {
				var vm = this;				
				if(data && data.properties) {
					Object.assign(vm.taskStatistic, data.properties);
				}
			},
			batchDeviceListLoadSuccess(data) {
				var vm = this;				
				if(data && data.properties) {
					Object.assign(vm.batchTaskDeviceStatistic, data.properties);
				}
			},
			//device list 表格数据加载成功回调
			deviceListLoadSuccess(data) {
				var vm = this;				
				if(data && data.properties) {
					Object.assign(vm.taskDeviceStatistic, data.properties);
				}
			},
			//task list 表格操作项
			optClick(row, ev) {
				var vm = this,
					status = row.TASK_STATUS;
				this.$root.rowData = row;

				if(vm.activeName == 'batch'){
					vm.menus= [										
		                {label:'<%=rb.getString("ZhongZhi")%>',cls:"el-icon el-icon-operation-terminate",code:'terminate'},								
						{label:'<%=rb.getString("XinXi")%>',cls:"el-icon el-icon-operation-info",code:'info'},
		                {label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete",code:'del'},
					];					
				}else{
					vm.menus= [					
						{label:'<%=rb.getString("KaiShi")%>',cls:"el-icon el-icon-operation-start",code:'start'},
		                {label:'<%=rb.getString("ZanTing")%>',cls:"el-icon el-icon-operation-awaiting",code:'stop'},
		                {label:'<%=rb.getString("ZhongZhi")%>',cls:"el-icon el-icon-operation-terminate",code:'terminate'},								
		                {label:'<%=rb.getString("XiuGai")%>',cls:"el-icon el-icon-operation-edit",code:'modify'},
						{label:'<%=rb.getString("XinXi")%>',cls:"el-icon el-icon-operation-info",code:'info'},
		                {label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete",code:'del'},
					];
				}

				if(!vm.hasConfigRole) {
					vm.menus= [						
						{label:'<%=rb.getString("XinXi")%>',cls:"el-icon el-icon-operation-info",code:'info'},
					];
				}
				
				initTaskStatus(status, vm.menus);
				vm.$nextTick(function(){
					document.body.click();
					if(vm.activeName == 'batch'){
						vm.$refs.batchMenu.show(ev);				
					}else{
						vm.$refs.menu.show(ev);
					}
					
				});
			},
			handerClose() {
				var vm = this;
				if(vm.activeName == 'batch'){
					this.$refs.batchMenu.hide();				
				}else{
					this.$refs.menu.hide();
				}
				
			},
			clickMenu(ev) {
				var vm = this,
					codes = {	
						start: vm.startTask,
						stop: vm.suspendTask,
						terminate: vm.terminateTask,
						modify: vm.editTask,
						info: vm.viewTask,						
						del: vm.deleteTask
					};

				if(codes[ev.code]){
    	    		codes[ev.code](this.$root.rowData)
    	    	}
			},
			//新建任务
			toAddTask(){
				var vm = this;
				vm.slideHeader = true;
	    	    vm.slideTitle = '<%=rb.getString("XinJianRenWu")%>';
	    	    vm.slideFooter = false;
	    	    vm.slidePosition = 'top';
	    	    vm.slideHeight = '100%';
	    	    vm.slideWidth = '100%';
				if(vm.activeName == 'batch'){
					vm.slideUrl = '${ctx}/cpe/batchconfig/toTaskInfoPage.action';			    	    
		    	    vm.$refs.slide.showSlide(function(){	    	    	
						eventBus.$emit("modify-task", vm.selection, 'add',vm.activeName );
		    	    });
				}else{
					vm.slideUrl = '${ctx}/cpe/factoryReset/toTaskInfoPage.action';			    	    
		    	    vm.$refs.slide.showSlide(function(){	    	    	
						eventBus.$emit("modify-task", vm.selection, 'add');
		    	    });
				}					    	   
			},
			//修改任务
			editTask(row) {
				var vm = this;
				vm.slideHeader = true;
				vm.slideTitle = '<%=rb.getString("XiuGai")%>';
				vm.slideUrl = '${ctx}/cpe/factoryReset/toTaskInfoPage.action?type=modify';				
				vm.slideFooter = false;
				vm.slidePosition = 'top'
	    	    vm.slideHeight = '100%'
	    	    vm.slideWidth = '100%'
				vm.$refs.slide.showSlide(function(){
					eventBus.$emit('modify-task', row, 'modify');
				});
			},
			//查看任务
			viewTask(row) {
				var vm = this;
				vm.slideHeader = true;
				vm.slideTitle = '<%=rb.getString("ChaKan")%>';
				//vm.slideUrl = '${ctx}/cpe/factoryReset/toTaskInfoPage.action?type=information';				
				vm.slideFooter = false;
				vm.slidePosition = 'top';
	    	    vm.slideHeight = '100%';
	    	    vm.slideWidth = '100%';
				/* vm.$refs.slide.showSlide(function(){
					eventBus.$emit('modify-task', row, 'information');
				}); */
				if(vm.activeName == 'batch'){
					vm.slideUrl = '${ctx}/cpe/batchconfig/toTaskInfoPage.action?type=information';			    	    
		    	    vm.$refs.slide.showSlide(function(){	    	    	
		    	    	eventBus.$emit('modify-task', row, 'information',vm.activeName);
		    	    });
				}else{
					vm.slideUrl = '${ctx}/cpe/factoryReset/toTaskInfoPage.action?type=information';			    	    
		    	    vm.$refs.slide.showSlide(function(){	    	    	
		    	    	eventBus.$emit('modify-task', row, 'information');
		    	    });
				}
			},
			/**
			 * 开始执行任务函数
			 * @param taskId: 当前数据ID
			*/
			startTask(row){ 
				var vm = this;
				axios.post('${ctx}/cpe/factoryReset/activeTask.action',stringify({
					taskId : row.TASK_ID,
				})).then(function(response){
					var data = response.data;
					if(data["success"]){
						vm.$refs.taskList.refresh();
						//是否需要提示
						vm.$message({
							type:'success',
							message:'<%=rb.getString("ChengGong")%>'
						})
					}else{
						vm.$message.error(data["message"]);
					}
				})
			},
			/**
			 * 暂停任务函数
			 * @param taskId: 当前数据ID
			*/
			suspendTask(row){
				var vm = this;
				axios.post('${ctx}/cpe/factoryReset/suspendTask.action',stringify({
					taskId : row.TASK_ID,
				})).then(function(response){
					var data = response.data;
					if(data["success"]){
						vm.$refs.taskList.refresh();
						//是否需要提示
						vm.$message({
							type:'success',
							message:'<%=rb.getString("ChengGong")%>'
						})
					}else{
						vm.$message.error(data["message"]);
					}
				})
			},
			/**
			 * 终止任务函数
			 * @param taskId: 当前数据ID
			*/
			terminateTask(row){ 
				var vm = this, terminateUrl = '';
				if(vm.activeName == 'batch'){
					terminateUrl = '${ctx}/cpe/batchconfig/terminateTask.action';
				}else {
					terminateUrl = '${ctx}/cpe/factoryReset/terminateTask.action';
				}
				axios.post(terminateUrl, stringify({
					taskId : row.TASK_ID,
				})).then(function(response){
					var data = response.data;
					if(data["success"]){
						if(vm.activeName == 'batch'){
							vm.$refs.batchTaskList.refresh();
						}else {
							vm.$refs.taskList.refresh();
						}
						vm.$message({
							type:'success',
							message:'<%=rb.getString("ChengGong")%>'
						})
					}else{
						vm.$message.error(data["message"]);
					}
				})
			},
			/**
			 *  删除任务
			 * @param taskId: 当前数据ID
			*/
			deleteTask(row){
				var vm = this, deleteUrl = '';
				if(vm.activeName == 'batch'){
					deleteUrl = '${ctx}/cpe/batchconfig/delTask.action';
				}else{
					deleteUrl = '${ctx}/cpe/factoryReset/delTask.action';
				}
				this.$confirm('<%=rb.getString("QueRenShanChuRenWu")%>',QueRen,{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					axios.post(deleteUrl, stringify({
						taskId:row.TASK_ID,
					})).then(function(response){
						var data = response.data;
						if(data["success"]){
							if(vm.activeName == 'batch'){
								vm.$refs.batchTaskList.refresh();
							}else {
								vm.$refs.taskList.refresh();
							}
							vm.$message({
								type:'success',
								message:'<%=rb.getString("ChengGong")%>'
							})
						}else{
							vm.$message.error(data["message"]);
						}
					}).catch(function(error){
						
					})
				}).catch()
			},			
			//batch config cpe list 搜索
			batchCpeListQuery(text) {
				var vm = this;
				vm.batchCpeListParams.search_text = text;
			},
			//cpe list 搜索
			cpeListQuery(text) {
				var vm = this;
				vm.cpeListParams.search_text = text;
			},
			
			batchTaskListQuery(text) {
				var vm = this;
				vm.batchTaskListParams.searchText = text;
			},
			//task list 搜索
			taskListQuery(text) {
				var vm = this;
				vm.taskListParams.searchText = text;
			},
			batchTaskListQueryAdvance() {
				var vm = this,
					times = vm.batchTaskListQueryForm.time;
				vm.batchTaskListParams.searchText = vm.batchTaskListQueryForm.searchText;
				vm.batchTaskListParams.startTime = times[0];
				vm.batchTaskListParams.endTime = times[1];
			},
			//task list 高级搜索
			taskListQueryAdvance() {
				var vm = this,
					times = vm.taskListQueryForm.time;
				vm.taskListParams.searchText = vm.taskListQueryForm.searchText;
				vm.taskListParams.startTime = times[0];
				vm.taskListParams.endTime = times[1];
			},
			batchTaskListReset() {
				var vm = this;
				Object.assign(vm.batchTaskListQueryForm, {
					searchText: '',
					time: []
				})
			},
			//task list 重置
			taskListReset() {
				var vm = this;
				Object.assign(vm.taskListQueryForm, {
					searchText: '',
					time: []
				})
			},
			batchDeviceListQuery(text) {
				var vm = this;
				vm.batchDeviceListParams.searchText = text;
			},
			//device list 搜索
			deviceListQuery(text) {
				var vm = this;
				vm.deviceListParams.searchText = text;
			},
			// device list 导出
			exportDevice() {
				var vm = this, exportUrl = '';
					
				if(vm.activeName == 'batch'){
					params = {
						timeZone: timeZone,
						searchText: vm.batchDeviceListParams.searchText
					};
					exportUrl = '${ctx}/cpe/batchconfig/exportDeviceListResult.action';
				}else {
					params = {
						timeZone: timeZone,
						searchText: vm.deviceListParams.searchText
					};
					exportUrl = '${ctx}/cpe/factoryReset/exportDeviceListResult.action';
				}
				exportByForm(exportUrl, params);
			},
			//结果字段格式化
			resultFmt(row, column, cellValue, index){
				var resultObj = {
						1: '<%=rb.getString("ChengGong")%>',
						2: '<%=rb.getString("ZhongZhi")%>',
						3: '<%=rb.getString("ShiBai")%>'
					};
				return resultObj[cellValue];
			},
			//是否保留配置格式化
			keepCongigFmt(row, column, cellValue, index){
				var keepCongigObj = {
						0: '<%=rb.getString("Fou")%>',
						1: '<%=rb.getString("Shi")%>'
					};
				return keepCongigObj[cellValue];
			},
			//关闭slide,更新表格数据
			cancelSlide() {
				var vm = this;								
				if(vm.activeName == 'batch'){
					vm.$refs.batchCpeList.clearSelection();
					vm.$refs.batchTaskList.refresh();
					vm.$refs.batchDeviceList.refresh();
				}else{
					vm.$refs.cpeList.clearSelection();
					vm.$refs.taskList.refresh();					
					vm.$refs.deviceList.refresh();
				}
				vm.$refs.slide.hide();
			},	
			dateChange(val) {
				var vm = this;
				vm.dateValue = val;
				if(val != null){
					vm.batchTaskListParams.startTime = vm.dateValue[0];
					vm.batchTaskListParams.endTime = vm.dateValue[1];
				}
			},
			// CPE 设备型号切换 4G/5G
			cpeModelClick(){
				var vm = this;
				vm.batchCpeListParams.cpe_model = vm.cpe_model;
				vm.$refs.batchCpeList.clearSelection();
			},
		},
		watch:{
			dateValue(newVal){
				var vm = this;
				if(!newVal){
					newVal = [];
					vm.batchTaskListParams.startTime = '';
	    			vm.batchTaskListParams.endTime = '';
				}
			},
		},
		mounted() {
			this.init();
			eventBus.$off('hide-slide').$on('hide-slide',this.cancelSlide);
		}
	})
</script>
