<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page import="com.baicells.omc.busi.system.login.entity.UserInfo" %>
<%@page import="com.baicells.omc.busi.utils.ComConstants" %>
<%
	UserInfo user = (UserInfo) session.getAttribute(ComConstants.SESSION_KEY);
%>
<style>
	#pciLockContent .queryInfo{
		display:inline-block;
		margin-right:80px;
	}
	#pciLockContent .queryInfo label{
		display:block;
	}
	#pciLockContent .el-date-editor--datetimerange.el-input__inner{
		width:340px;
	}
	#pciLockContent .el-query .el-form-item{
		display:inline-block;
		margin-right:80px;
	}
	
	#pciLockContent .advanceQuery{
		height:28px;
	}
	#pciLockContent .el-icon-common-query-down,
	#pciLockContent .arrow-text,
	#pciLockContent .el-icon-common-query-up{
		line-height:30px;
	}
	#pciLockContent .pairgrid-query .el-input__inner{
		height:26px;
		line-height:26px;
	}
	#pciLockContent .el-icon-operation-lock:before{
		color:#F2B354;
	}
	#pciLockContent .suc_count span,
	#pciLockContent .fail_count span{
		height:18px;
		line-height:18px;
		padding: 0 10px;
		display:inline-block;
		font-size:14px;
		vertical-align:bottom;
	}
	#pciLockContent .suc_count i,
	#pciLockContent .fail_count i{
		margin-right:5px;
		font-size:16px;
	}
	#pciLockContent .suc_count,
	#pciLockContent .fail_count{
		display:inline-block;
		margin: 0;
		padding: 0;
	}
	#pciLockContent .suc_count span:first-child{
		color:#67D972 !important;
	}
	#pciLockContent .suc_count span:last-child{
		color:#333;
		font-weight:normal;
		margin: 0;
	}
	#pciLockContent .fail_count span:first-child{
		border-right:0px;
		color:#E88282 !important;
	}
	#pciLockContent .fail_count span:last-child{
		color:#333;
		font-weight:normal;
		margin: 0;
	}
	#pciLockContent .suc_count .el-icon-circle-success:before{
		color:#67D972
	}
	#pciLockContent .fail_count .el-icon-circle-close:before{
		color:#E88282
	}
	
	#pciLockContent .commonTabsTop .el-tabs__header{
		border: 1px solid #D5DCEC;
		border-radius: 8px 8px 0 0;
	}
</style>
<!-- cpe pci lock -->
<div class="overflow-cls">
<div class='panelDefault' id='pciLockContent' style="min-width: 1000px;position: relative; width: 100%; height: 100%;border: 0; background: unset;">
	<div v-show="addButton" class='circleIcon placeholder-bt' :placeholder="placeText" :style='styleObj' style='top: 6px;'> 
		<span class='el-icon' :class='buttonIcon' @click='addPciTask'></span>
	</div>
	
	<el-tabs v-model='activeName' style='height:calc(100% - 2px);' @tab-click='tabClick' class='commonTabsTop'>
		<el-tab-pane label='eNB' name='enb' class="CODE_ENB_MONITOR hidden visible ">
			<el-ctable ref="enb_table" :url='enbPciUrl' time=6 :query-params="enb_params" :rownumber="true" id="pci_table" height="100%" page-size=20 pagination="true" class='commonWarp commonTableBorder'>
				<template slot="toolbar">
					<el-query @query="query" @advance-query="advanceQuery" @reset="resetQuery" placeholder="<%=rb.getString("PCIRenWuMing")%>/<%=rb.getString("ChuangJianZhe")%>" ok-text="<%=rb.getString("ChaXun")%>" reset-text="<%=rb.getString("ChaXunChongZhi")%>">
						<template slot="form">
							<div class='queryInfo'>
								<label><%=rb.getString("PCIRenWuMing")%></label>
								<el-input v-model='enb_params.taskName' size="mini" style='width:200px'></el-input>
							</div>
							<div class='queryInfo'>
								<label><%=rb.getString("ZhuangTai")%></label>
								<el-select v-model='enb_params.taskStatus'>
									<el-option v-for='item in statusOptions' :label='item.label' :value='item.value'></el-option>
								</el-select>
							</div>
							<div class='queryInfo'>
								<label><%=rb.getString("ChuangJianZhe")%></label>
								<el-input v-model='enb_params.userCode' size="mini" style='width:200px'></el-input>
							</div>
							<div class='queryInfo'>
								<label><%=rb.getString("JieGuo")%></label>
								<el-select v-model='enb_params.taskResult'>
									<el-option v-for='item in resultOptions' :label='item.label' :value='item.value'></el-option>
								</el-select>
							</div>
							<div class='queryInfo'>
								<label><%=rb.getString("KaiShiShiJian")%></label>
								<el-date-picker v-model='enbDateVal' size="mini"  value-format="yyyy-MM-dd HH:mm:ss" type="datetimerange" range-separator="——"  start-placeholder='<%=rb.getString("KaiShiShiJian")%>' end-placeholder='<%=rb.getString("JieShuShiJian")%>'></el-date-picker>
							</div>
						</template>
					</el-query>
				</template>
				<el-table-column label='' width="30" class-name="no-text-tips">
					<template slot-scope="scope">
	            		<div class="el-icon el-icon-operation-more" @click="enbOptClick(scope.row,event)" v-clickoutside="handerClose" style="cursor: pointer;"></div>
	          		</template>
				</el-table-column>
				<el-table-column label='<%=rb.getString("RenWuMingCheng")%>' width="300" prop="TASK_NAME"></el-table-column>
				<el-table-column label='<%=rb.getString("ChuangJianZhe")%>' width="150" prop="CREATE_USER"></el-table-column>
				<el-table-column label='<%=rb.getString("KaiShiShiJian")%>' width="200" prop="START_TIME"></el-table-column>
				<el-table-column label='<%=rb.getString("JieShuShiJian")%>' width="200" prop="END_TIME"></el-table-column>
				<el-table-column label='<%=rb.getString("ZhuangTai")%>' width="100" prop="TASK_STATUS">
					<template slot-scope="scope">
						<div v-html="taskTableStatus(scope.row.TASK_STATUS)"></div>
					</template>
				</el-table-column>
				<el-table-column label='<%=rb.getString("JinDu")%>' width="200" prop="TASK_PROGRESS" ></el-table-column>
				<el-table-column label='<%=rb.getString("JieGuo")%>' prop="TASK_RESULT" :formatter='resultFmt'></el-table-column>
			</el-ctable>
			<el-cmenu ref="enbMenu" :data="enbMenus" @click="enbClickMenu"></el-cmenu>
		</el-tab-pane>
		
		<el-tab-pane label='CPE' name='cpe' class="CODE_CPE_PCI_LOCK hidden visible">
			<div style="flex:1;overflow:auto;" class='commonWarp commonTableBorder'>
				<el-ctable ref="cpe_list_table" id="cpe_list_table" row-key="CPE_CODE" :url="cpeUrl" :height="height" :query-params="query_cpe_params" pagination="true" @selection-change="selectChangeCpe">
					<el-table-column type="selection" width="45" reserve-selection=true v-if="showChecked"></el-table-column>
					<el-table-column prop="connection_status" width="50">
						<template slot-scope="scope">
							<div :class="{
							'el-icon el-icon-status-conn-off':scope.row.CONNECTION_STATUS!='Exception' && scope.row.CONNECTION_STATUS!='On' && scope.row.CONNECTION_STATUS!='updating' && scope.row.CONNECTION_STATUS!=1,
							'':scope.row.have_connected==2,
							'conn_exc':scope.row.CONNECTION_STATUS=='Exception',
							'el-icon el-icon-status-conn-on':scope.row.CONNECTION_STATUS=='On'||scope.row.CONNECTION_STATUS=='updating'||scope.row.CONNECTION_STATUS==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
						</template>
					</el-table-column>
					<el-table-column prop='SERIAL_NUMBER' label='<%=rb.getString("CPEXuLieHao")%>' width="200"></el-table-column>
					<el-table-column prop='CPE_NAME' label='<%=rb.getString("CPEName")%>' width="150"></el-table-column>
					<el-table-column prop='IMSI' label='IMSI' width="150"></el-table-column>
					<el-table-column label='<%=rb.getString("SheBeiXingHao")%>' prop="cpe_model" key="cpe_model"  min-width="120"></el-table-column>
					<el-table-column prop='SCANMODE' label='<%=rb.getString("SaoMiaoFangShi")%>' width="120" :formatter='modeFmt'></el-table-column>
					<el-table-column prop='DL_EARFCN' label='<%=rb.getString("PinDian")%>' width="100">
						<template slot-scope='scope'>
							<div style="display: flex;" v-html="earfcnStatus(scope.row.DL_EARFCN,scope.row)"></div>
						</template>
					</el-table-column>
					<el-table-column prop='PCI' label='PCI' width="100">
						<template slot-scope='scope'>
							<div style="display: flex;" v-html="pciStatus(scope.row.PCI,scope.row)"></div>
						</template>
					</el-table-column>
					<el-table-column prop='MODEL_NAME' label='<%=rb.getString("ChanPinXingHao")%>' width="150"></el-table-column>
					<el-table-column prop='MACADDRESS' label='<%=rb.getString("CPEMacAddress")%>' width="150"></el-table-column>
					<el-table-column prop='IPADDRESS' label='<%=rb.getString("CPEIPAddress")%>' width="150"></el-table-column>
					<el-table-column prop='group_name' label='<%=rb.getString("SheBeiZu")%>'></el-table-column>
					<template slot='toolbar' style="position:relative;">
						<el-form :model='query_cpe_form' ref="query_cpe_form" label-position="top">
							<span style='font-size:14px;font-weight:bold;position:absolute;top:18px;left:20px;'><%=rb.getString("SuoPinCPELieBiao")%></span>
							<el-query @query="queryCpe" @advance-query="advanceQueryCpe" @reset='resetQueryCpe' :placeholder="'<%=rb.getString("CPEXuLieHao")%>/<%=rb.getString("CPEName")%>/IMSI/PCI'"
							:ok-text="'<%=rb.getString("ChaXun")%>'" :reset-text="'<%=rb.getString("ChaXunChongZhi")%>'" style='margin-left:70px;'>
								<template slot="form">
									<el-form-item class='deviceItem' label='<%=rb.getString("CPEXuLieHao")%>' prop='serial_number'>
										<el-input v-model='query_cpe_form.serial_number'  size="mini"></el-input>
									</el-form-item>
									<el-form-item class='deviceItem' label='<%=rb.getString("CPEName")%>' prop='CPE_NAME'>
										<el-input v-model='query_cpe_form.CPE_NAME' size="mini"></el-input>
									</el-form-item>
									<el-form-item class='deviceItem' label='<%=rb.getString("PinDian")%>' prop='earfcn'>
										<el-input v-model='query_cpe_form.earfcn' size="mini"></el-input>
									</el-form-item>
									<el-form-item class='deviceItem' label='PCI' prop='pci'>
										<el-input v-model='query_cpe_form.pci' size="mini"></el-input>
									</el-form-item>
									<el-form-item class='deviceItem' label='<%=rb.getString("SheBeiZu")%>' prop='group_id'>
										<el-select v-model="query_cpe_form.group_id" size="mini">
											<el-option v-for="item in groupOptions" :key="item.value" :label="item.text" :value="item.value">
											</el-option>
										</el-select>
									</el-form-item>
								</template>
							</el-query>
						</el-form>
						<el-radio-group size="mini" v-model='cpe_model' class="commonRadioButton" @change="cpeModelClick" style='width:155px;position:absolute;right:20px;top:10px;'>
							<el-radio-button label="4G">4G CPE</el-radio-button>
							<el-radio-button label="5G">5G CPE</el-radio-button>
						</el-radio-group>
					</template>
				</el-ctable>
			</div>
			<div style="flex:1;overflow:auto; margin-top: 10px;" class=" ">
				<el-tabs v-model="listName" style='height:100%'>
					<el-tab-pane label="<%=rb.getString("RenWuLieBiao")%>" name="taskList">
						<el-ctable ref="cpe_table" time=6 :query-params="cpe_params" :rownumber="true" id="pci_task_table" :url="cpePciUrl" height="100%" page-size=20 pagination="true"
							style='border: 1px solid #D5DCEC;border-top:none;height:calc(100% - 2px); border-radius: 0 0 8px 8px;'>
							<template slot="toolbar">
								<el-form :model='cpe_form' ref="cpe_form" label-position="top">
									<el-query @query="query" @advance-query="advanceQuery" @reset="resetQuery" placeholder="<%=rb.getString("PCIRenWuMing")%>/<%=rb.getString("ChuangJianZhe")%>" ok-text="<%=rb.getString("ChaXun")%>" reset-text="<%=rb.getString("ChaXunChongZhi")%>">
										<template slot="form">
											<el-form-item class='queryInfo' label="<%=rb.getString("PCIRenWuMing")%>" prop="taskName">
												<el-input v-model='cpe_form.taskName' size="mini" style='width:200px'></el-input>
											</el-form-item>
											<el-form-item class='queryInfo' label="<%=rb.getString("ZhuangTai")%>" prop="taskStatus">
												<el-select v-model='cpe_form.taskStatus'>
													<el-option v-for='item in statusOptions' :label='item.label' :value='item.value'></el-option>
												</el-select>
											</el-form-item>
											<el-form-item label="<%=rb.getString("ChuangJianZhe")%>" prop="userCode">
												<el-input v-model='cpe_form.userCode' size="mini" style='width:200px'></el-input>
											</el-form-item>
											<el-form-item label="<%=rb.getString("JieGuo")%>" prop="taskResult">
												<el-select v-model='cpe_form.taskResult'>
													<el-option v-for='item in resultOptions' :label='item.label' :value='item.value'></el-option>
												</el-select>
											</el-form-item>
											<el-form-item label="<%=rb.getString("KaiShiShiJian")%>">
												<el-date-picker v-model='cpeDateVal' size="mini"  value-format="yyyy-MM-dd HH:mm:ss" type="datetimerange" range-separator="——"  start-placeholder='<%=rb.getString("KaiShiShiJian")%>' end-placeholder='<%=rb.getString("JieShuShiJian")%>'></el-date-picker>
											</el-form-item>
										</template>
									</el-query>
								</el-form>
								
							</template>
							<el-table-column label='' width="30" class-name="no-text-tips">
								<template slot-scope="scope">
				            		<div class="el-icon el-icon-operation-more" @click="cpeOptClick(scope.row,event)" v-clickoutside="handerClose" style="cursor: pointer;"></div>
				          		</template>
							</el-table-column>
							<el-table-column label='<%=rb.getString("RenWuMingCheng")%>' width="400" prop="TASK_NAME"></el-table-column>
							<el-table-column label='<%=rb.getString("CaoZuoRen")%>' width="150" prop="CREATE_USER"></el-table-column>
							<el-table-column label='<%=rb.getString("CaoZuoShiJian")%>' width="150" prop="CREATE_TIME"></el-table-column>
							<el-table-column label='<%=rb.getString("SaoMiaoFangShi")%>' width="150" prop="LOCK_MODE" :formatter='modeFmt'></el-table-column>
							<el-table-column label='5G <%=rb.getString("SaoMiaoFangShi")%>' width="120" prop='NR_LOCK_MODE'></el-table-column>
							<el-table-column label='<%=rb.getString("ZhuangTai")%>' width="150" prop="TASK_STATUS">
								<template slot-scope="scope">
									<div v-html="taskTableStatus(scope.row.TASK_STATUS)"></div>
								</template>
							</el-table-column>
							<el-table-column label='<%=rb.getString("JinDu")%>' width="80" prop="TASK_PROGRESS"></el-table-column>
							<el-table-column label='<%=rb.getString("JieGuo")%>' prop="TASK_RESULT" :formatter='resultFmt'></el-table-column>
							<el-table-column label='<%=rb.getString("KaiShiShiJian")%>' width="200" prop="START_TIME"></el-table-column>
							<el-table-column label='<%=rb.getString("JieShuShiJian")%>' width="200" prop="END_TIME"></el-table-column>
						</el-ctable>
						<el-cmenu ref="cpeMenu" :data="cpeMenus" @click="cpeClickMenu"></el-cmenu>
					</el-tab-pane>
					<el-tab-pane label="<%=rb.getString("BackupRestoreSheBeiLieBiao")%>" name="deviceList">
						<el-ctable ref="cpe_result_table" id="cpe_result_table" time=6 :url="cpeResultUrl" :height="height" :query-params="cpe_result_params" pagination="true" @load-success="loadsuccessResult"
						style='border: 1px solid #D5DCEC;border-top:none;height:calc(100% - 2px); border-radius: 0 0 8px 8px;'>
							<el-table-column prop='SERIAL_NUMBER' label='<%=rb.getString("CPEXuLieHao")%>' width="200"></el-table-column>
							<el-table-column prop='CPE_NAME' label='<%=rb.getString("CPEName")%>' width="200"></el-table-column>
							<el-table-column prop='IMSI' label='IMSI' width="150"></el-table-column>
							<el-table-column prop='TASK_NAME' label='<%=rb.getString("RenWuMingCheng")%>' width="300"></el-table-column>
							<el-table-column prop='LOCK_MODE' label='<%=rb.getString("SaoMiaoFangShi")%>' width="100" :formatter='modeFmt'></el-table-column>
							<el-table-column prop='NR_LOCK_MODE' label='5G <%=rb.getString("SaoMiaoFangShi")%>' width="120"></el-table-column>
							<el-table-column prop='EARFCN' label='<%=rb.getString("PinDian")%>' width="100"></el-table-column>
							<el-table-column prop='PCI' label='PCI' width="100"></el-table-column>
							<el-table-column prop='NR_EARFCN' label='5G <%=rb.getString("PinDian")%>' width="100"></el-table-column>
							<el-table-column prop='NR_PCI' label='5G PCI' width="100"></el-table-column>
							<el-table-column prop='NR_BAND' label='Band' width="100"></el-table-column>
							<el-table-column prop="PROGRESS_STATUS" label='<%=rb.getString("ZhuangTai")%>' width="120" >
								<template slot-scope="scope">
									<div v-html="taskTableStatus(scope.row.PROGRESS_STATUS)"></div>
								</template>
							</el-table-column>
							<el-table-column prop="PROGRESS_RESULT" label='<%=rb.getString("JieGuo")%>' width="100" :formatter="deviceResultFmt"></el-table-column>
							<el-table-column prop='FAILURE_REASON' label='<%=rb.getString("ShiBaiYuanYin")%>' width="200"></el-table-column>
							<el-table-column prop='START_TIME' label='<%=rb.getString("KaiShiShiJian")%>' width="200"></el-table-column>
							<el-table-column prop='END_TIME' label='<%=rb.getString("JieShuShiJian")%>' width="200"></el-table-column>
							<template slot="toolbar">
								<div class='queryGroup' style='height:26px;'>
									<el-input v-model='cpe_result_form.searchText' @keyup.enter.native="queryResult" class='pairgrid-query' placeholder='<%=rb.getString("CPEBianMa")%> / <%=rb.getString("CPEName")%> / <%=rb.getString("RenWuMingCheng")%>'></el-input>
									<i @click='queryResult' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
								</div>
								<div style='float:right;margin-right:50px; margin-top: 5px;'>
									<p class='suc_count'><span><i class='el-icon el-icon-circle-success'></i><%=rb.getString("ChengGong")%></span><span>{{sucNum}}</span></p>
									<p class='fail_count'><span><i class='el-icon el-icon-circle-close'></i><%=rb.getString("ShiBai")%></span><span>{{failNum}}</span></p>
								</div>
								<div class="circleIcon placeholder-bt" style="top:10px;right:15px;" placeholder="<%=rb.getString("DaoChu")%>" @click="exportResult">		
									<span class="el-icon el-icon-circle-export"></span>
								</div>
							</template>
						</el-ctable>
					</el-tab-pane>
				</el-tabs>
			</div>
			<el-bulk v-if="false" ref="viewCpeBulk" target="cpe_list_table" :list="selectionCpe" row-key="CPE_CODE"  show-prop="SERIAL_NUMBER"
				:message="{title:'<%=rb.getString("YiXuanSheBei")%>',subTitle:'<%=rb.getString("CPEXuLieHao")%>',clear:'<%=rb.getString("QingKong")%>',cancel:'<%=rb.getString("QuXiao")%>'}">
				<template slot="button">
					<el-dropdown placement='top-start' trigger='click' style='float:left;margin-right:15px;' @command='handleCommand'>
						<el-button type='primary' size='small' style='height:24px;'><%=rb.getString("YouXiaoQiSuoDing")%><i class='el-icon-arrow-down' style='font-size:10px;color:#fff;margin-left:5px;'></i></el-button>
						<el-dropdown-menu slot='dropdown'>
							<el-dropdown-item command='freqpreferred'>Frequency Preferred</el-dropdown-item>
							<el-dropdown-item command='pcilock'>PCI Lock</el-dropdown-item>
							<el-dropdown-item command='pcionlylock'>PCI Only Lock</el-dropdown-item>
						</el-dropdown-menu>
					</el-dropdown>
					<a class="linkbutton linkbutton_trend" @click="handleCommand('fullband')"><span><%=rb.getString("JieSuo")%></span></a>
				</template>
			</el-bulk>
		</el-tab-pane>
	</el-tabs>
	
	<!-- 二级页面 -- 查看旧版本日志 -->
	<el-slide ref="slide" :url="slideUrl" :title="slideTitle" class='commonBorderSlide'
			  :footer="slideFooter" :header='slideHeader' :position="slidePosition" :height="slideHeight" :modal='slideModal'  :width="slideWidth" :subloading="slideSubmitLoading" @ok='saveTask' @cancel='cancelTask' :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'">
	 </el-slide>
	 <el-slide ref="exportSlide" :url="exportUrl" title="<%=rb.getString("ZhiXingJieGuo")%>" :footer="false" header='true' position="bottom"
	 height="310px" modal='false'  width="100%"  @cancel='cancelExport' >
	 	<template slot='toolbar'>
	 		<a href='#' @click='exportMsg' style='position:absolute;right:45px;top:0px;' class='el-icon el-icon-operation-export'></a>
	 	</template>
	 </el-slide>
</div>
</div>
<script type="text/javascript">
	var pciLockVue = new Vue({
		el:'#pciLockContent',
		data:{
			activeName : 'enb',
			enb_params:{
				timeZone:timeZone,
				taskName:'',
				taskStatus:'',
				taskResult:'',
				userCode:'',
				startTime:'',
				endTime:''
			},
			cpe_params:{
				timeZone:timeZone,
				taskName:'',
				taskStatus:'',
				taskResult:'',
				userCode:'',
				startTime:'',
				endTime:'',
				searchText:''
			},
			cpe_form:{
				taskName:'',
				taskStatus:'',
				taskResult:'',
				userCode:'',
				startTime:'',
				endTime:''
			},
			statusOptions:[
				{
					label:'<%=rb.getString("QuanBu")%>',
					value:''
				},
				{
					label:'<%=rb.getString("DengDai")%>',
					value:'1'
				},
				{
					label:'<%=rb.getString("JinXingZhong")%>',
					value:'2'
				},
				{
					label:'<%=rb.getString("ZanTing")%>',
					value:'3'
				},
				{
					label:'<%=rb.getString("YiJieShu")%>',
					value:'4'
				}
			],
			resultOptions:[
				{
					label:'<%=rb.getString("QuanBu")%>',
					value:''
				},
				{
					label:'<%=rb.getString("ChengGong")%>',
					value:'1'
				},
				{
					label:'<%=rb.getString("BuFenChengGong")%>',
					value:'2'
				},
				{
					label:'<%=rb.getString("ShiBai")%>',
					value:'3'
				}
			],
			slideUrl:'',
			slideHeight:'100%',
			slideWidth:'',
            slideSubmitLoading:'',
			slideModal:false,
			slideHeader:true,
			slideFooter:false,
			slidePosition:'',
			slideTitle:'',
			styleObj:{
		    	zIndex:10
		    },
		    buttonIcon:'el-icon-circle-add',
		    placeText:'<%=rb.getString("TianJia")%>',
			enbPciUrl:'${ctx}/task/pcilock/getPciLockTaskList.action',
			cpePciUrl:'${ctx}/cpe/strategy/getPciLockTaskList.action',
			enbMenus:[],
			cpeMenus:[],
			rowData:[],
			rowDataCpe:[],
			addEnbFlag:true,
			addCpeFlag:true,
			enbOperType:'',
			cpeOperType:'',
			exportUrl:'',
			enbDateVal:[],
			cpeDateVal:[],
			addButton:true,
			height:'100%',
			query_cpe_params:{
				timeZone:timeZone,
				search_text:"",
				like_fields : 'CPE_NAME,IMSI,serial_number,pci',
				serial_number:'',
				CPE_NAME:'',
				earfcn:'',
				pci:'',
				group_id:'',
				cpe_model:'4G'
			},
			query_cpe_form:{
				serial_number:'',
				CPE_NAME:'',
				earfcn:'',
				pci:'',
				group_id:''
			},
			cpeUrl:'${ctx}/cell/CPE/queryCpeInfosList.action?forSelect=9',
			groupOptions:[],
			listName:"taskList",
			cpeDateVal:[],
			cpeMenus:[],
			selectionCpe:[],
			cpeResultUrl:'${ctx}/cpe/strategy/getPciLockTaskDeviceList.action',
			cpe_result_params:{timeZone:timeZone,searchText:''},
			cpe_result_form:{searchText:''},
			sucNum:'0',
			failNum:'0',
			showChecked:true,
			cpe_model:'4G'
		},
		methods:{
			init(){
				var vm = this;
				if(writableMap["CODE_ENB_MONITOR"] != undefined){
					this.activeName = "enb"
					if(writableMap["CODE_ENB_MONITOR"]){
						this.addButton = true;
					}else{
						this.addButton = false;
					}
				}else if(writableMap["CODE_CPE_PCI_LOCK"] != undefined){
					this.activeName = "cpe"
					if(writableMap["CODE_CPE_PCI_LOCK"]){
						this.addButton = true;
					}else{
						this.addButton = false;
					}
				}
				axios.post("${ctx}/cell/CPE/getDeviceGroup4Combobox.action",stringify({
					"operator_code":operator_code
				})).then(function(response){
					var data = response.data;
					if ( data ){
						vm.groupOptions = data;
					}
				})
				vm.showChecked = writableMap["CODE_CPE_PCI_LOCK"] ? true : false;
			},
			//点击页面其他地方菜单收起
			handerClose(){
				if(this.activeName == 'enb'){
    	        	this.$refs.enbMenu.hide();
				}else{
					this.$refs.cpeMenu.hide();
				}
    	    },
			/**
			 * 获取  扫描方式 数据项 进行转化
			 * @param row:当前行数据
			 * @param column:当前行数据的dom
			 * @param value:当前行的传入值
			 * @param index：下标
			*/
    	    modeFmt(row,column,value,index){
    	    	if(value == 'fullband'){
    	    		return "<%= rb.getString("BuSuoPin")%>";
    	    	}else if(value == 'pcilock'){
    	    		return "<%= rb.getString("SuoPCI")%>";
    	    	}else if(value == 'freqpreferred'){
    	    		return "<%= rb.getString("SuoPin")%>";
    	    	}else if(value == 'pcionlylock'){
    	    		return "PCI only lock";
    	    	}
    	    },
			/**
			 * 获取结果数据项 进行转化
			 * @param row:当前行数据
			 * @param column:当前行数据的dom
			 * @param value:当前行的传入值
			 * @param index：下标
			*/
    	    resultFmt(row,column,value,index){
    	    	if (value == "1") {
    				return "<%= rb.getString("ChengGong")%>";
    			} else if (value == "2") {
    				return "<%= rb.getString("BuFenChengGong")%>";
    			} else if (value == "3") {
    				return "<%= rb.getString("ShiBai")%>";
    			} else {
    				return "";
    			}
    	    },
    	    deviceResultFmt(row,column,value,index) {
    	    	if (value == "3") {
					return ShiBai;
				} else if (value == "1") {
					return ChengGong;
				} else if (value == "2") {
					return ZhongZhi;
				}else {
					return "";
				}
    	    },
			/**
			 * 点击更多操作函数
			 * @param row:当前行数据
			 * @param ev:event
			*/
    	    enbOptClick(row,ev){
    	    	var vm = this;
    	    	vm.rowData = row;
    	    	var task_status = row.TASK_STATUS;
    	    	vm.enbMenus = [
    	    		{label:'<%=rb.getString("JieGuo")%>',code:'view'},
			        {label:'<%=rb.getString("KaiShi")%>',cls:"CODE_CPE_PCI_LOCK hidden",code:'start'},
			        {label:'<%=rb.getString("ZanTing")%>',cls:"CODE_CPE_PCI_LOCK hidden" ,code:'stop'},
			        {label:'<%=rb.getString("ZhongZhi")%>',cls:"CODE_CPE_PCI_LOCK hidden",code:'end'},
					{label:'<%=rb.getString("XinXi")%>',code:'info'},
			        {label:'<%=rb.getString("XiuGai")%>',cls:"CODE_CPE_PCI_LOCK hidden",code:'edit'},
			        {label:'<%=rb.getString("ShanChu")%>',cls:"CODE_CPE_PCI_LOCK hidden",code:'del'}
    	    	]
    	    	initTaskStatus(task_status,vm.enbMenus);
    	    	vm.$nextTick(function(){
		    		document.body.click();
    		    	vm.$refs.enbMenu.show(ev);
		    	});
    	    },
			/**
			 * eNb点击事件的获取值
			 * @param ev:点击值进行筛选赋值
			*/
    	    enbClickMenu(ev){
    	    	var codes = {
    	    			view:this.viewResult,
    	    			start:this.activeEnbTask,
    	    			stop:this.suspendEnbTask,
    	    			end:this.terminateEnbTask,
    	    			info:this.viewEnbTask,
	    	    		edit:this.editEnbTask,
	    	    		del:this.delEnbTask
	    	    }
    	    	if(codes[ev.code]){
    	    		codes[ev.code]()
    	    	}
    	    },
			/**
			 * cpe 点击更多下拉菜单
			 * @param row:当前行的值
			*/
    	    cpeOptClick(row,ev){
    	    	var vm = this;
    	    	vm.rowDataCpe = row;
    	    	var task_status = row.TASK_STATUS;
    	    	vm.cpeMenus = [
			        {label:'<%=rb.getString("KaiShi")%>',cls:"CODE_CPE_PCI_LOCK hidden",code:'start'},
			        {label:'<%=rb.getString("ZanTing")%>',cls:"CODE_CPE_PCI_LOCK hidden" ,code:'stop'},
			        {label:'<%=rb.getString("ZhongZhi")%>',cls:"CODE_CPE_PCI_LOCK hidden",code:'end'},
			        {label:'<%=rb.getString("XinXi")%>',code:'info'},
			        {label:'<%=rb.getString("XiuGai")%>',cls:"CODE_CPE_PCI_LOCK hidden",code:'edit'},
			        {label:'<%=rb.getString("ShanChu")%>',cls:"CODE_CPE_PCI_LOCK hidden",code:'del'}
    	    	]
    	    	initTaskStatus(task_status,vm.cpeMenus);
    	    	vm.$nextTick(function(){
		    		document.body.click();
    		    	vm.$refs.cpeMenu.show(ev);
		    	});
    	    },
			/**
			 * cpe点击事件的获取值
			 * @param ev:点击值进行筛选赋值
			*/
    	    cpeClickMenu(ev){
    	    	var codes = {
    	    			view:this.viewResult,
    	    			start:this.activeCpeTask,
    	    			stop:this.suspendCpeTask,
    	    			end:this.terminateCpeTask,
    	    			info:this.viewCpeTask,
	    	    		edit:this.editCpeTask,
	    	    		del:this.delCpeTask
	    	    }
    	    	if(codes[ev.code]){
    	    		codes[ev.code]()
    	    	}
    	    },
			/**
			 * 模糊查询
			 * @param val：传入参数 进行查询
			*/
			query(val){
				if(this.activeName == 'enb'){
					this.resetQuery();
	    			this.enb_params.searchText  = val;
	    			this.$refs.enb_table.refresh()
				}else{
	    			this.cpe_params.searchText  = val;
				}
			},
			// 高级查询 确定按钮
			advanceQuery(){
				if(this.activeName == 'enb'){
					this.enb_params.searchText = "";
	    			if(this.enbDateVal != null){
	    				this.enb_params.startTime = this.enbDateVal[0];
		    			this.enb_params.endTime = this.enbDateVal[1];
	    			}
	    			this.$refs.enb_table.refresh()
				}else{
	    			if(this.cpeDateVal != null){
	    				this.cpe_form.startTime = this.cpeDateVal[0];
		    			this.cpe_form.endTime = this.cpeDateVal[1];
	    			}
	    			Object.assign(this.cpe_params,this.cpe_form);
				}
			},
			// 普通搜索 查询按钮
			resetQuery(){
				if(this.activeName == 'enb'){
					this.enb_params.taskName = '';
					this.enb_params.taskStatus = '';
					this.enb_params.taskResult = '';
					this.enb_params.userCode = '';
	    			this.enbDateVal = []
				}else{
	    			this.cpeDateVal = []
	    			this.$refs.cpe_form.resetFields();
	    			this.cpe_form.startTime = '';
	    			this.cpe_form.endTime = '';
				}
			},
			addPciTask(){//点击添加按钮
				var vm = this;
				vm.$refs.exportSlide.hide();
				if(vm.activeName == 'enb'){
					if(vm.addEnbFlag){
			           	vm.slideTitle = '<%=rb.getString("XinJianPinDianSuoRenWu")%>';
			           	vm.slideUrl = '${ctx}/task/pcilock/goAddTask.action';
			           	vm.slideHeader=true;
			           	vm.slideFooter=true;
						vm.slidePosition='top';
						vm.slideWidth='100%';
                        vm.slideSubmitLoading = false;
			    		vm.buttonIcon = 'el-icon-circle-close'
			    	    vm.placeText='<%=rb.getString("GuanBi")%>';
			    		vm.addEnbFlag = false;
			    		vm.enbOperType = 'add';
						vm.$refs.slide.showSlide(function(){
				    	    vm.slideModal = false
				    	});
					}else{
						vm.cancelEnbSlide();
					}
				}else{
					if(vm.addCpeFlag){
			           	vm.slideTitle = '<%=rb.getString("XinJianPinDianSuoRenWu")%>';
			           	vm.slideUrl = '${ctx}/cpe/strategy/toModifyTaskInfoPage.action';
			           	vm.slideHeader=true;
			           	vm.slideFooter=true;
						vm.slidePosition='top';
						vm.slideWidth='100%';
                        vm.slideSubmitLoading = false;
			    		vm.buttonIcon = 'el-icon-circle-close'
			    	    vm.placeText='<%=rb.getString("GuanBi")%>';
			    		vm.addCpeFlag = false;
			    		vm.cpeOperType='add';
						vm.$refs.slide.showSlide(function(){
							eventBus.$emit('get-cpe-info','','add');
				    	    vm.slideModal = false
				    	});
					}else{
						vm.cancelCpeSlide();
					}
				}
			},
			// 关闭展开enb
			cancelEnbSlide(){
				eventBus.$emit('cancel-add-enb');
			},
			// 关闭展开cpe
			cancelCpeSlide(){
				eventBus.$emit('cancel-add-cpe');
			},
			// cpe结果按钮函数
			viewResult(){
				var vm = this;
				if(vm.activeName == 'enb'){
					vm.exportUrl = "${ctx}/task/pcilock/toPciLockTaskProgress.action?task_id=" + vm.rowData.TASK_ID;
				}else{
					vm.exportUrl = "${ctx}/cpe/strategy/toPciLockTaskProgress.action?task_id=" + vm.rowDataCpe.TASK_ID;
				}
				vm.$refs.exportSlide.showSlide();
			},
			// enb开始函数
			activeEnbTask(){
				var vm = this;
    	    	axios.post('${ctx}/task/pcilock/activeTask.action',stringify({
    	    		taskId : vm.rowData.TASK_ID
    	    	})).then(function(response){
    	    		var data = response.data;
    	    		if(data["success"]){
    	    			vm.$refs.enb_table.refresh()
    	    		}else{
    	    			vm.$message.error(data["message"])
    	    		}
    	    	})
			},
			// enb 暂停函数
			suspendEnbTask(){
				var vm = this;
    	    	axios.post('${ctx}/task/pcilock/suspendTask.action',stringify({
    	    		taskId : vm.rowData.TASK_ID
    	    	})).then(function(response){
    	    		var data = response.data;
    	    		if(data["success"]){
    	    			vm.$refs.enb_table.refresh()
    	    		}else{
    	    			vm.$message.error(data["message"])
    	    		}
    	    	})
			},
			// enb终止函数
			terminateEnbTask(){
				var vm = this;
    	    	axios.post('${ctx}/task/pcilock/terminatePciLockTask.action',stringify({
    	    		taskId : vm.rowData.TASK_ID
    	    	})).then(function(response){
    	    		var data = response.data;
    	    		if(data["success"]){
    	    			vm.$refs.enb_table.refresh()
    	    		}else{
    	    			vm.$message.error(data["message"])
    	    		}
    	    	})
			},
			// enb删除函数
			delEnbTask(){
				var vm = this;
    	    	this.$confirm('<%=rb.getString("QueRenShanChuRenWu")%>',QueRen,{
    	    		customClass:'warningConfirm',
    	    		confirmButtonText:'<%=rb.getString("QueDing")%>',
    	    		cancelButtonText:'<%=rb.getString("QuXiao")%>',
    	    		type:'warning',
    	    		closeOnClickModal:false
    	    	}).then(() => {
    	    		axios.post('${ctx}/task/pcilock/delPciLockTask.action',stringify({
    		    		taskId:vm.rowData.TASK_ID
    		    	})).then(function(response){
    		    		var data = response.data;
    		    		if(data["success"]){
    		    			vm.$refs.enb_table.refresh()
    		    			vm.$message({
    			    			type:'success',
    			    			message:'<%=rb.getString("ShanChuChengGong")%>'
    			    		})
							vm.$refs.exportSlide.hide();
    		    		}else{
    		    			vm.$message.error(data["message"])
    		    		}
    		    	}).catch(function(error){
    		    		
    		    	})
    	    	}).catch()
			},
			// enb函数
			viewEnbTask(){
				var vm = this;
				vm.slideUrl = '${ctx}/task/pcilock/toInformationTaskInfoPage.action?taskId=' + vm.rowData.TASK_ID;
				vm.slideHeader = true;
				vm.slideFooter = false;
				vm.slideTitle = '<%=rb.getString("XinXi")%>';
				vm.slidePosition = 'left';
				vm.slideWidth = '95%';
                vm.slideSubmitLoading = false;
				vm.styleObj = {
		   	    	zIndex:10
		   	    }
				vm.enbOperType = 'view';
				vm.$refs.slide.showSlide(function(){
		    	    vm.slideModal = true;
		    	    eventBus.$emit('view-enb-info');
		    	});
			},
			editEnbTask(){
				var vm = this;
				vm.slideUrl = '${ctx}/task/pcilock/toModifyTaskInfoPage.action?taskId=' + vm.rowData.TASK_ID + '&type=edit';
				vm.slideHeader = true;
				vm.slideFooter = true;
				vm.slideTitle = '<%=rb.getString("XiuGai")%> eNB PCI Lock';
				vm.slidePosition = 'left';
				vm.slideWidth = '95%';
                vm.slideSubmitLoading = false;
				vm.enbOperType = 'edit';
				vm.styleObj = {
			   	   	zIndex:10
			   	}
				vm.$refs.slide.showSlide(function(){
		    	    vm.slideModal = true;
		    	    eventBus.$emit('get-enb-info');
		    	});
			},
			// cpe开始函数
			activeCpeTask(){
				var vm = this;
    	    	axios.post('${ctx}/cpe/strategy/activeTaskForCpe.action',stringify({
    	    		taskId : vm.rowDataCpe.TASK_ID
    	    	})).then(function(response){
    	    		var data = response.data;
    	    		if(data["success"]){
    	    			vm.$refs.cpe_table.refresh()
    	    		}else{
    	    			vm.$message.error(data["message"])
    	    		}
    	    	})
			},
			// cpe暂停函数
			suspendCpeTask(){
				var vm = this;
    	    	axios.post('${ctx}/cpe/strategy/suspendTaskForCpe.action',stringify({
    	    		taskId : vm.rowDataCpe.TASK_ID
    	    	})).then(function(response){
    	    		var data = response.data;
    	    		if(data["success"]){
    	    			vm.$refs.cpe_table.refresh()
    	    		}else{
    	    			vm.$message.error(data["message"])
    	    		}
    	    	})
			},
			// cpe停止函数
			terminateCpeTask(){
				var vm = this;
    	    	axios.post('${ctx}/cpe/strategy/terminatePciLockTask.action',stringify({
    	    		taskId : vm.rowDataCpe.TASK_ID
    	    	})).then(function(response){
    	    		var data = response.data;
    	    		if(data["success"]){
    	    			vm.$refs.cpe_table.refresh()
    	    		}else{
    	    			vm.$message.error(data["message"])
    	    		}
    	    	})
			},
			// cpe删除函数
			delCpeTask(){
				var vm = this;
    	    	this.$confirm('<%=rb.getString("QueRenShanChuRenWu")%>',QueRen,{
    	    		customClass:'warningConfirm',
    	    		confirmButtonText:'<%=rb.getString("QueDing")%>',
    	    		cancelButtonText:'<%=rb.getString("QuXiao")%>',
    	    		type:'warning',
    	    		closeOnClickModal:false
    	    	}).then(() => {
    	    		axios.post('${ctx}/cpe/strategy/delPciLockTask.action',stringify({
    		    		taskId:vm.rowDataCpe.TASK_ID
    		    	})).then(function(response){
    		    		var data = response.data;
    		    		if(data["success"]){
    		    			vm.$refs.cpe_table.refresh()
    		    			vm.$message({
    			    			type:'success',
    			    			message:'<%=rb.getString("ShanChuChengGong")%>'
    			    		})
    		    		}else{
    		    			vm.$message.error(data["message"])
    		    		}
    		    	}).catch(function(error){
    		    		
    		    	})
    	    	}).catch()
			},
			// cpe信息函数
			viewCpeTask(){
				var vm = this;
				vm.slideUrl = '${ctx}/cpe/strategy/toInformationTaskInfoPage.action';
				vm.slideHeader = true;
				vm.slideFooter = false;
				vm.slideTitle = '<%=rb.getString("XinXi")%>';
				vm.slidePosition = 'left';
				vm.slideWidth = '85%';
                vm.slideSubmitLoading = false;
				vm.cpeOperType = 'view';
				vm.styleObj = {
			   	    	zIndex:10
			   	}
				vm.$refs.slide.showSlide(function(){
		    	    vm.slideModal = true;
		    	    eventBus.$emit('get-cpe-info',vm.rowDataCpe.TASK_ID,'view');
		    	});
			},
			// cpe 修改函数
			editCpeTask(){
				var vm = this;
				vm.slideUrl = '${ctx}/cpe/strategy/toModifyTaskInfoPage.action';
				vm.slideHeader = true;
				vm.slideFooter = true;
				vm.slideTitle = '<%=rb.getString("XiuGai")%> CPE PCI Lock';
				vm.slidePosition = 'left';
				vm.slideWidth = '95%';
                vm.slideSubmitLoading = false;
				vm.cpeOperType = 'edit';
				vm.styleObj = {
			   	    	zIndex:10
			   	}
				vm.$refs.slide.showSlide(function(){
		    	    vm.slideModal = true;
		    	    eventBus.$emit('get-cpe-info',vm.rowDataCpe.TASK_ID,'modify');
		    	});
			},
			// 关闭新建窗口 判断是哪个tab切换的
			cancelTask(){
				var vm = this;
				if(vm.activeName == 'enb'){
					if(vm.enbOperType == 'view'){
						vm.cancelViewEnb();
					}else{
						vm.cancelEnbSlide();
					}
				}else{
					if(vm.cpeOperType == 'view'){
						vm.cancelViewCpe();
					}else{
						vm.cancelCpeSlide();
					}
				}
			},
			//  关闭Enb信息窗口
			cancelViewEnb(){
				this.$refs.slide.hide();
			},
			//  关闭Cpe信息窗口
			cancelViewCpe(){
				this.$refs.slide.hide();
			},
			// 修改和新建cpe 或 enb 任务
			saveTask(){
				var vm = this;
				if(vm.activeName == 'enb'){
					eventBus.$emit('add-enb');
				}else{
					eventBus.$emit('add-cpe');
				}
			},
			// 关闭enb新建窗口
			closeEnb(){
				var vm = this;
				if(vm.enbOperType == 'add'){
		        	vm.styleObj = {
			   	    		zIndex:10
			   	    }
		    		vm.buttonIcon = 'el-icon-circle-add'
			    	vm.placeText='<%=rb.getString("TianJia")%>'
			    	vm.addEnbFlag = true;
				}
				vm.$refs.slide.hide();
			},
			// 关闭cpe新建窗口
			closeCpe(){
				var vm = this;
				if(vm.cpeOperType == 'add'){
		        	vm.styleObj = {
			   	    		zIndex:10
			   	    }
		    		vm.buttonIcon = 'el-icon-circle-add'
			    	vm.placeText='<%=rb.getString("TianJia")%>'
			    	vm.addCpeFlag = true;
				}
				vm.$refs.slide.hide();
			},
			// 保存enb新建
			saveEnbSuccess(){
				var vm = this;
				if(vm.enbOperType == 'add'){
		    		vm.buttonIcon = 'el-icon-circle-add'
			    	vm.placeText='<%=rb.getString("TianJia")%>'
			    	vm.addEnbFlag = true
				}
		    	vm.$refs.slide.hide();
		    	vm.$refs.enb_table.refresh()
			},
			// 保存cpe新建
			saveCpeSuccess(){
				var vm = this;
				if(vm.cpeOperType == 'add'){
		    		vm.buttonIcon = 'el-icon-circle-add'
			    	vm.placeText='<%=rb.getString("TianJia")%>'
			    	vm.addCpeFlag = true
				}
		    	vm.$refs.slide.hide();
		    	vm.$refs.cpe_table.refresh()
			},
			// 导出执行结果 enb或者cpe
			exportMsg(){
				if(this.activeName == 'enb'){
					eventBus.$emit('export-enb');
				}else{
					eventBus.$emit('export-cpe');
				}
			},
			//关闭执行结果 弹窗
			cancelExport(){
				this.$refs.exportSlide.hide();
			},
			/**
			 * tab点击当前行
			 * @param tab:当前行 数据
			*/
			tabClick(tab){
				this.$refs.exportSlide.hide();
				if(this.activeName == "enb"){
					if(writableMap["CODE_ENB_MONITOR"]){
						this.addButton = true;
					}else{
						this.addButton = false;
					}
				}else if(this.activeName == "cpe"){
					if(writableMap["CODE_CPE_PCI_LOCK"]){
						this.addButton = true;
					}else{
						this.addButton = false;
					}
				}
			},
			queryCpe(val){
				this.query_cpe_params.search_text = val;
			},
			advanceQueryCpe(){
				Object.assign(this.query_cpe_params,this.query_cpe_form);
			},
			resetQueryCpe(){
				this.$refs.query_cpe_form.resetFields();
			},
			selectChangeCpe(selection){
				this.selectionCpe = selection;
			},
			pciStatus(value, rowData) {
				var reg = new RegExp("^(IDU\/CN)");
				if("LTE WiFi VoIP Gateway" == rowData["OLDPRODUCT"] || (reg.test(rowData["OLDPRODUCT"])==true) ){
					return "--";
				}
				if(!value) return "";
				var pciValue = rowData.PCI;
				if(rowData.SCANMODE == 'pcilock' || rowData.SCANMODE == 'pcionlylock' ){
					var imgL = "<div class='pciClass'><span class='el-icon el-icon-operation-lock easyui-tooltip' style='font-size:16px;'></span></div><span style='margin-left:5px;'>"+pciValue+"</span>" ;
					return imgL;
				}else{
					var imgL = "<div class='pciClass'><span class='el-icon el-icon-status-unlock easyui-tooltip'></span></div><span>"+pciValue+"</span>" ;
					return imgL;
				}
			},
			earfcnStatus(value,rowData){
				if(rowData.SCANMODE == 'fullband' || rowData.SCANMODE == 'pcionlylock'){
					var imgL = "<div class='pciClass'><span class='el-icon el-icon-status-unlock easyui-tooltip'></span></div><span>"+value+"</span>" ;
					return imgL;
				}else if(rowData.SCANMODE == 'freqpreferred' || rowData.SCANMODE == 'pcilock'){
					var imgL = "<div class='pciClass'><span class='el-icon el-icon-operation-lock easyui-tooltip' style='font-size:16px;'></span></div><span style='margin-left:5px;'>"+value+"</span>" ;
					return imgL;
				}else{
					return value;
				}
			},
			loadsuccessResult(data){
				this.sucNum = data.properties.success;
		    	this.failNum = data.properties.fail;
			},
			exportResult(){
				var vm = this;
		    	var params = {
		    	    timeZone: timeZone,
		    		searchText : vm.cpe_result_form.searchText
		    	}
		    	var url = "${ctx}/cpe/strategy/exportPciLockDeviceListToCSV.action";
		    	exportByForm(url,params);
			},
			queryResult(){
				Object.assign(this.cpe_result_params,this.cpe_result_form);
			},
			handleCommand(val){
				var vm = this;
				var cpeCodes = vm.selectionCpe.map(function(item){
					return item.CPE_CODE;
				})
				var params = {
					cpeCodes : cpeCodes.toString(),
					lockType : val
				}
				vm.$confirm('<%=rb.getString("QueDingXiuGaiSuoPingFangShi")%>','<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					axios.post("${ctx}/cpe/strategy/updatePciLock.action",stringify(params)).then(function(response){
						var data = response.data;
						if(data["success"]){
							vm.$message({
								message:"<%=rb.getString("ChengGong")%>",
								type:'success'
							})
							vm.$refs.cpe_list_table.refresh();
							vm.$refs.cpe_list_table.clearSelection();
						}
					});
				}).catch(() => {})
			},
			// CPE 设备型号切换 4G/5G
			cpeModelClick(){
				var vm = this;
				vm.query_cpe_params.cpe_model = vm.cpe_model;
				vm.$refs.cpe_list_table.clearSelection();
			},
		},
		mounted(){
			this.init();
			eventBus.$off('save-enb').$on('save-enb',this.saveEnbSuccess);
			eventBus.$off('save-cpe').$on('save-cpe',this.saveCpeSuccess);
			eventBus.$off('close-enb').$on('close-enb',this.closeEnb);
			eventBus.$off('close-cpe').$on('close-cpe',this.closeCpe);
			eventBus.$off('close-edit-enb').$on('close-edit-enb',this.closeEditEnb);
		}
	})
</script>