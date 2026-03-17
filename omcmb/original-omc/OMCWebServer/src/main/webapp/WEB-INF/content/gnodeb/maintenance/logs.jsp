<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page import="com.baicells.omc.busi.system.login.entity.UserInfo" %>
<%@page import="com.baicells.omc.busi.utils.ComConstants" %>
<%
	UserInfo user = (UserInfo) session.getAttribute(ComConstants.SESSION_KEY);
%>

<style>
#gnbLogs .el-tabs__nav-wrap::after {
	right:120px;
}
#gnbLogs .el-pagination .el-select .el-input{
	width:85px;
}
#gnbLogs .el-pagination .el-select .el-input .el-input__inner{
	width:85px;
	background:#fff !important;
}

#gnbLogs .h100{height: 100%}
#gnbLogs .curpo{
	cursor: pointer
}
#gnbLogs .el-icon-status-terminate:before,
#gnbLogs .el-icon-status-waiting1:before,
#gnbLogs .el-icon-status-inProgress:before{
	color:#4D84FF;
}
#gnbLogs .el-icon-status-success:before {
	color:#67D972;
}
#gnbLogs .el-icon-status-failed:before {
	color:#E88282;
}
#gnbLogs .curStatus{
	font-size:18px;
	margin-right:5px;
}

#gnbLogs .exceptionLogOptCls .el-icon::before{
	font-size: 18px;
}
#gnbLogs .exceptionLogOptCls .logsOptBoxCls{
	display: flex;
	align-items: center;
}
#gnbLogs .exceptionLogOptCls .logsOptBoxCls > div{
	margin-left: 10px;
	position: relative;
	cursor: pointer;
}
#gnbLogs .el-tabs__header{
	border: 1px solid #E4E7EC;
}
#gnbLogs .el-tabs__item{
	height: 36px;
	line-height: 36px;
}
#gnbLogs .el-ctable-toolbar{
	padding: 0px!important;
}
#gnbLogs .commonQuery .el-date-editor .el-range__close-icon{
	line-height: 20px;
}
#gnbLogs .logsOptBtnBoxCls .el-button{
	border-color: transparent;
    color: #4D84FF;
	padding: 0px;
	background: transparent;
	font-size: 12px;
}
#gnbLogs .logsOptBtnBoxCls .el-button.is-loading:before{
	background: transparent;
}
#gnbLogs .logsOptBtnBoxCls .el-button .el-icon-loading::before{
    font-weight: 600;
    font-size: 18px;
}
#gnbLogs .logsOptBoxCls .excCollectFiedIconCls{
    position: absolute;
    top: 5px;
    left: 7px;
    height: 14px;
    width: 14px;
    border-radius: 7px;
    background-color: #FFF;
}
#gnbLogs .logsOptBoxCls .excCollectFiedIconCls > span{
    height: 14px;
    width: 14px;
    line-height: 14px;
    text-align: center;
}
#gnbLogs .logsOptBoxCls .excCollectFiedIconCls .el-icon::before{
	color: red;
	font-size: 14px;
}
</style>
<div class="overflow-cls">
<!-- 设备日志功能页面 -->
<div class="panelDefault overHide commonWarp" id='gnbLogs' style="min-width: 900px;position: relative;overflow: hidden;">
	<!-- 按钮  -- 新建任务 -->
	<div class="newIconBoxCls-bt CODE_GNB_LOGS hidden" style="right:10px;top:5px;" v-show="activeName == 'first'" placeholder="<%=rb.getString("TianJia")%>" @click="collectLogs">
		<span class="el-icon-circle-add el-icon"></span>
	</div>
	<!--设备异常日志-导出-->
    <div class="newIconBoxCls-bt CODE_GNB_LOGS hidden" style="right:10px;top:5px;" v-show="activeName == 'third'" placeholder="<%=rb.getString("DaoChu")%>">
        <span class="el-icon el-icon-circle-export" @click="exceptionExportLogs"></span>
    </div>
	<!-- 事件日志-- 统计 -->
	<div class="newIconBoxCls-bt" style="right:45px;top:5px;" v-show="activeName == 'forth'" placeholder="<%=rb.getString("TongJi")%>">
		<span class="el-icon el-icon-circle-statistic" @click="statisEventLog"></span>
	</div>
	<!-- 事件日志-导出 -->
	<div class="newIconBoxCls-bt" style="right:10px;top:5px;" v-show="activeName == 'forth'"  placeholder="<%=rb.getString("DaoChu")%>">
		<span class="el-icon el-icon-circle-export" @click="exportLogs"></span>
	</div>

	<!-- 主页面区域 -- tab页 -->
	<el-tabs v-model="activeName" @tab-click='tabClick' class="h100" v-show="tabsShow">
		<!-- 设备上报日志 -->
		<el-tab-pane label="<%=rb.getString("DeviceSheBeiRiZhi")%>" name="first" class=''>
			<el-ctable id="immeLogFileGnbTaskDatagrid" ref="ctableReport" :time="6" url="${ctx}/cell/collect/getImmediateCollectLogTaskPageList.action" :height="height" 
				:row-key="'serial_number'" :query-params="params_report" pagination="true" :rownumber=true @selection-change='batchSelect'>
				<!-- 高级查询 -- 设备上报日志-->
				<template slot="toolbar">
					<div v-if="false" class='toolbarHeadBtnBoxCls'>
						<!-- 已选数据 -->
						<div class="selectBlukBoxCls">
							<div class="selectMain headBtnItemCls">
								<div class="bulkSelectBtnBoxCls"  @click="openBulkSelectTable" style='border-right: 0; padding: 0;'>
									<span class="el-icon-selected el-icon"></span>
									<span class="bulkSelectNumBoxCls">( {{selectedList.length}} )</span>
								</div>
								<div class="selectTableBoxCls" v-show="bulkDeviceSelectShow" style="position: absolute;top: 32px;left: 30px;">
									<div class="selectBoxTitle">
										<span><%=rb.getString("YiXuan")%></span>
										<span style="position:absolute;right:20px;top:15px;" class="el-icon el-icon-close" @click="closeBulkSelectTable"></span>
									</div>
									<div class="selectBoxMain">
										<div class="tableInfoCls">
											<div class="tableInfoHeader">
												<div><%=rb.getString("YiXuanWenJian")%></div>
												<div @click="clearBulkSelected"><span style="margin-right:5px;" class="el-icon el-icon-operation-delete" ></span><%=rb.getString("QingChu")%></div>
											</div>
											<el-ctable 
												id="gnbDeviceBulkDeviceSelectTable" 
												ref="gnbDeviceBulkDeviceSelectTable" 
												:data="selectedList" 
												:showHeader="false"
												:rownumber="false"
												:front-pagination="true"
												height="270px" pagination="true" >
												<el-table-column prop="id" v-if="false"></el-table-column>
												<el-table-column width="588">
													<template slot-scope="scope" >
														<div class="tableItemCls">
															<span>{{scope.row.serial_number}}</span>
															<span @click="delBulkSelected(scope.row)" class="el-icon el-icon-circle-close item_show"></span>
														</div>
													</template>
												</el-table-column>
											</el-ctable>
										</div>
									</div>
								</div>
							</div>
						</div>
		
						<div :class="selectedList.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="batchDeleteForLog" style='border-right: 0;'>
							<span class='el-icon el-icon-operation-delete'></span>
							<span><%=rb.getString("ShanChu")%></span>
						</div>
						<div :class="selectedList.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="downloadImmediateLogFile" style='border-right: 0;'>
							<span class='el-icon el-icon-operation-delete'></span>
							<span><%=rb.getString("XiaZai")%></span>
						</div>
					</div>
					<div class='commonQuery' style='display: flex;align-items: center;height:45px;'>
						<el-query type="normal" @query="queryDevice" placeholder="<%=rb.getString("SheBeiWeiYiBiaoZhi")%>"></el-query>  
						<el-date-picker style='margin-left: 20px;' 
							v-model="queryDeviceDateTime"
							type="datetimerange"
							value-format="yyyy-MM-dd HH:mm:ss"
							range-separator="——"  
							@change="queryDeviceDateTimeChange"
							start-placeholder='<%=rb.getString("KaiShiShiJian")%>' 
							end-placeholder='<%=rb.getString("JieShuShiJian")%>'>
						</el-date-picker>            
					</div>
				</template>
				
				<el-table-column label='' width="50" type="selection" v-if="false"></el-table-column>
				<!-- 主列表 -->
				<el-table-column label='' width="30" prop="" class-name="no-text-tips">
					<template slot-scope="scope">
	            		<div class="el-icon el-icon-operation-more curpo" @click="optClick(scope.row,event)" v-clickoutside="handerClose"></div>
	          		</template>
				</el-table-column>
				<el-table-column label='<%=rb.getString("SheBeiWeiYiBiaoZhi")%>' min-width="200" prop="serial_number"></el-table-column>
				
 				<el-table-column prop="task_status" label="<%=rb.getString("ShouJiZhuangTai")%>" min-width="220">
					<template slot-scope="scope">
						<!-- 立即执行的任务-状态：0-4；周期执行的任务-状态： 0-8-->					
						<div v-if="scope.row.execute_type == 'Immediately'">	
							<!-- 0 等待 ：可终止 ，可删除-->					
							<div v-if="scope.row.task_status == 0">
								<span class="el-icon el-icon-status-waiting1 curStatus"></span>
								<span><%=rb.getString("DengDai")%></span>
							</div>
							<!--1 进行中：可终止，不可删除-->
							<div v-if="scope.row.task_status == 1">
								<span class="el-icon el-icon-status-inProgress curStatus"></span>
								<span><%=rb.getString("JinXingZhong")%></span>
							</div>
							<!-- 2 成功-->
							<div v-if="scope.row.task_status == 2">
								<span class="el-icon el-icon-status-success curStatus"></span>
								<span><%=rb.getString("ChengGong")%></span>
							</div>
							<!-- 3 失败-->
							<div v-if="scope.row.task_status == 3">
								<span class="el-icon el-icon-status-failed curStatus"></span>
								<span><%=rb.getString("LogsShiBai")%></span>
							</div>
							<!-- 4 终止-->
							<div v-if="scope.row.task_status == 4">
								<span class="el-icon el-icon-status-terminate curStatus"></span>
								<span><%=rb.getString("ZhongZhi")%></span>
							</div>
						</div>	
							
						<div v-else>
							<div v-if="scope.row.task_status == 0">
								<span class="el-icon el-icon-status-waiting1 curStatus"></span>
								<span><%=rb.getString("DengDai")%></span>
							</div>
							<div v-if="scope.row.task_status == 1">
								<span class="el-icon el-icon-status-inProgress curStatus"></span>
								<span><%=rb.getString("JinXingZhong")%></span>
							</div>
							<div v-if="scope.row.task_status == 2">
								<span class="el-icon el-icon-status-success curStatus"></span>
								<span><%=rb.getString("ChengGong")%></span>
							</div>
							<div v-if="scope.row.task_status == 3">
								<span class="el-icon el-icon-status-failed curStatus"></span>
								<span><%=rb.getString("LogsShiBai")%></span>
							</div>
							<div v-if="scope.row.task_status == 4">
								<span class="el-icon el-icon-status-terminate curStatus"></span>
								<span><%=rb.getString("ZhongZhi")%></span>
							</div>
							<!-- 5 正在停止上报  （进行中）：可终止，不可删除  (实际已在终止中，不可再点击终止操作)-->
							<div v-if="scope.row.task_status == 5">
								<span class="el-icon el-icon-status-inProgress curStatus"></span>
								<span><%=rb.getString("ZhongZhiZhong")%></span>
							</div>
							<!-- 6  停止上报 成功   （终止）-->
							<div v-if="scope.row.task_status == 6">
								<span class="el-icon el-icon-status-terminate curStatus"></span>
								<span><%=rb.getString("ZhongZhi")%></span>
							</div>
							<!-- 7 停止上报失败  （ 进行中）：可终止，不可删除 -->
							<div v-if="scope.row.task_status == 7">
								<span class="el-icon el-icon-status-inProgress curStatus"></span>
								<span><%=rb.getString("JinXingZhong")%></span>
							</div>
							<!-- 8 周期上报设置成功，重启生效（等待）：可终止，不可删除  -->
							<div v-if="scope.row.task_status == 8">
								<span class="el-icon el-icon-status-waiting1 curStatus"></span>
								<span><%=rb.getString("ZhouQiShangBaoSheZhiChengGongCQSX")%></span>
							</div>
						</div>
										
					</template>
				</el-table-column>				
				<el-table-column label='<%=rb.getString("ZiKaiZhanZhiXingFangShi")%>' min-width="120"  prop="execute_type" ></el-table-column>
 				<el-table-column label='<%=rb.getString("WenJianShuLiang")%>' min-width="120" prop="file_num" v-if='false'></el-table-column>
				<el-table-column label='<%=rb.getString("ShiBaiYuanYin")%>' min-width="200" prop="failureReason" show-overflow-tooltip="true"></el-table-column>
				<el-table-column label='<%=rb.getString("LogsGengXinShiJian")%>' min-width="150" prop="update_time"></el-table-column>

			</el-ctable>
			<el-cmenu ref="menuReport" :data="menus" @click="clickMenu"></el-cmenu>
		</el-tab-pane>

		<!-- 设备异常日志 -->
		<el-tab-pane label="<%=rb.getString("DeviceSheBeiYiChangRiZhi")%>" name="third" class=''>
			<el-ctable id="gnbDeviceErrorLogFileDatagrid" ref="ctableException" :height="height" :url="'${ctx}/system/logmange/device/getExceptionLogPageList.action'"  @selection-change='batchSelect'
				:row-key="'id'" :time="6" :query-params="params_exception" @load-success="loadSuccessExcTable" pagination="true" :rownumber=true>
				
				<!-- 高级查询 -- 异常日志 -->
				<template slot="toolbar">
					<div class='toolbarHeadBtnBoxCls' v-show="isWritable">
						<!-- 已选数据 -->
						<div class="selectBlukBoxCls">
							<div class="selectMain headBtnItemCls">
								<div class="bulkSelectBtnBoxCls"  @click="openBulkSelectTable" style='border-right: 0; padding: 0;'>
									<span class="el-icon-selected el-icon"></span>
									<span class="bulkSelectNumBoxCls">( {{selectedList.length}} )</span>
								</div>
								<div class="selectTableBoxCls" v-show="bulkExcSelectShow" style="position: absolute;top: 32px;left: 30px;">
									<div class="selectBoxTitle">
										<span><%=rb.getString("YiXuan")%></span>
										<span style="position:absolute;right:20px;top:15px;" class="el-icon el-icon-close" @click="closeBulkSelectTable"></span>
									</div>
									<div class="selectBoxMain">
										<div class="tableInfoCls">
											<div class="tableInfoHeader">
												<div><%=rb.getString("YiXuanWenJian")%></div>
												<div @click="clearBulkSelected"><span style="margin-right:5px;" class="el-icon el-icon-operation-delete" ></span><%=rb.getString("QingChu")%></div>
											</div>
											<el-ctable 
												id="gnbExcBulkExcSelectTable" 
												ref="gnbExcBulkExcSelectTable" 
												:data="selectedList" 
												:showHeader="false"
												:rownumber="false"
												:front-pagination="true"
												height="270px" pagination="true" >
												<el-table-column prop="id" v-if="false"></el-table-column>
												<el-table-column width="588">
													<template slot-scope="scope" >
														<div class="tableItemCls">
															<span>{{scope.row.serial_number}}</span>
															<span @click="delBulkSelected(scope.row)" class="el-icon el-icon-circle-close item_show"></span>
														</div>
													</template>
												</el-table-column>
											</el-ctable>
										</div>
									</div>
								</div>
							</div>
						</div>
		
						<div :class="selectedList.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="batchDeleteForLog" style='border-right: 0;'>
							<span class='el-icon el-icon-operation-delete'></span>
							<span><%=rb.getString("ShanChu")%></span>
						</div>
					</div>
					<div class='commonQuery' style='display: flex;align-items: center;height:45px;'>
						<el-query type="normal" @query="queryExc" placeholder="<%=rb.getString("SheBeiWeiYiBiaoZhi")%>"></el-query>  
						<el-date-picker style='margin-left: 20px;' 
							v-model="queryExcDateTime"
							type="datetimerange"
							value-format="yyyy-MM-dd HH:mm:ss"
							range-separator="——"  
							@change="queryExcDateTimeChange"
							start-placeholder='<%=rb.getString("KaiShiShiJian")%>' 
							end-placeholder='<%=rb.getString("JieShuShiJian")%>'>
						</el-date-picker>
						<el-popfilter style="margin-left: 20px;"
							type="single"
							label='<%=rb.getString("YiChangLeiXing") %>'
							v-model="query_exception.operate_type"
							:list="exceptionTypeOptions.map(item=>{return {label:item.text,value:item.id}})"
							@check-change="advanceQuery">
						</el-popfilter>
					</div>
				</template>
	
				<el-table-column label='' width="50"  type="selection" v-if="isWritable" :selectable="checkSelectTable"></el-table-column>
				<!-- 主列表 -->
				<el-table-column label='' width="100" prop="" class-name="no-text-tips exceptionLogOptCls">
					<template slot-scope="scope">
						<div v-if="scope.row.manual_collection_status != '1'" class="logsOptBoxCls">

							<div v-if="scope.row.file_name && scope.row.manual_collection_status != '2' && scope.row.is_file_deleted == '0'" class="el-icon el-icon-collected disabled" title='<%=rb.getString("KaiShiShouJi") %>'></div>
							<div v-if="isWritable && ((!scope.row.file_name && scope.row.manual_collection_status != '2') || (scope.row.file_name && scope.row.is_file_deleted == '1' && scope.row.manual_collection_status != '2'))" class="el-icon el-icon-collected" @click="collectLogFile(scope.row,event)" title='<%=rb.getString("KaiShiShouJi") %>'></div>
							<div v-if="isWritable && scope.row.manual_collection_status == '2'" @click="collectLogFile(scope.row,event)" title='<%=rb.getString("KaiShiShouJi") %>'>
								<span class="el-icon el-icon-collected"></span>
								<div class="excCollectFiedIconCls">
									<span class="el-icon el-icon-circle-warning"></span>
								</div>
							</div>
							<div v-if="!scope.row.file_name || scope.row.is_file_deleted == '1'" class="el-icon el-icon-operation-download disabled" title='<%=rb.getString("XiaZai") %>'></div>
							<div v-if="isWritable && scope.row.file_name && scope.row.is_file_deleted == '0'" class="el-icon el-icon-operation-download" @click="downlodFile(scope.row,'Task')" title='<%=rb.getString("XiaZai") %>'></div>
							<div v-if="isWritable" class="el-icon el-icon-operation-delete" @click="delCollectFile(scope.row,'Task')" title='<%=rb.getString("ShanChu") %>'></div>
						</div>
						<div v-if="scope.row.manual_collection_status == '1'" class="logsOptBtnBoxCls">
							<el-button :loading="scope.row.manual_collection_status == '1'">Collecting</el-button>
						</div>
	          		</template>
				</el-table-column>
				<el-table-column label='<%=rb.getString("SheBeiWeiYiBiaoZhi")%>' min-width="200" prop="serial_number"></el-table-column>
				<el-table-column label='<%=rb.getString("Title_SheBeiMingCheng")%>' width="200"  prop="name" ></el-table-column>
				<el-table-column label='<%=rb.getString("Title_SheBeiLeixing")%>' width="150" prop="device_type" ></el-table-column>
				<el-table-column label='<%=rb.getString("JiZhanIP")%>' width="200" prop="operate_ip" ></el-table-column>
				<el-table-column label='<%=rb.getString("ChanPinLeiXing")%>' width="150" prop="product" ></el-table-column>
				<el-table-column label='<%=rb.getString("RuanJianBanBen")%>' width="200" prop="software_version" ></el-table-column>
				<el-table-column label='<%=rb.getString("YiChangLeiXing")%>' width="150" prop="operation_name"></el-table-column>
				<el-table-column label='<%=rb.getString("WenJianMing")%>' min-width="200" prop="file_name">
					<template slot-scope="scope">
						<div v-if="scope.row.file_name && scope.row.manual_collection_status != '2'" :style="{'color':(scope.row.is_file_deleted == '1' ? '#9A9B9D' : '')}">{{scope.row.file_name}}</div>
						<div v-if="scope.row.manual_collection_status == '2'" style="color:#FF4614;">{{scope.row.collection_fail_reason}}</div>
					</template>
				</el-table-column>
				<el-table-column label='<%=rb.getString("ShiJian")%>' width="160" prop="op_start_time"></el-table-column>
				<el-table-column label='<%=rb.getString("YunXingShiJian")%>' width="160" prop="runtime_before_reboot"></el-table-column>
                <el-table-column label='<%=rb.getString("SiJiYuanYin")%>' width="160" prop="halt_detail_reason"></el-table-column>
			</el-ctable>
		</el-tab-pane>
		
		<!-- 事件日志 -->
		<el-tab-pane v-if="false" label="<%=rb.getString("ShiJianRiZhi")%>" name="forth" class=''>
			<el-ctable id="gnbEventLogLists" ref="ctableEvent" url="${ctx}/cell/cellEventLog/getCellEventPageList.action" :height="height" 
					   :query-params="params_event" pagination="true" :rownumber="true">
				
				<!-- 查询 -- 事件日志 -->
				<template slot="toolbar">
					<div class='commonQuery' style='display: flex;align-items: center;height:45px;'>
						<el-query type="normal" @query="queryEvent" placeholder="ID / <%=rb.getString("SheBeiWeiYiBiaoZhi")%> / <%=rb.getString("JiZhanIP")%>"></el-query>  
						<el-date-picker style='margin-left: 20px;' 
							v-model="queryEventDateTime"
							type="datetimerange"
							value-format="yyyy-MM-dd HH:mm:ss"
							range-separator="——"  
							@change="queryEventDateTimeChange"
							start-placeholder='<%=rb.getString("KaiShiShiJian")%>' 
							end-placeholder='<%=rb.getString("JieShuShiJian")%>'>
						</el-date-picker>
						<el-popfilter style="margin-left: 20px;"
							type="single"
							label='<%=rb.getString("ShiJianLeiXing") %>'
							v-model="query_event.eventName"
							:list="statisticOptions.map(item=>{return {label:item.text,value:item.id}})"
							@check-change="advanceQuery">
						</el-popfilter>
					</div>
				</template>
				
				<!-- 主列表 -->
				<el-table-column label='ID' width="60" prop="id"></el-table-column>
				<el-table-column label='<%=rb.getString("SheBeiWeiYiBiaoZhi")%>' min-width="200" prop="ne_code"></el-table-column>
				<el-table-column label='<%=rb.getString("JiZhanIP")%>' min-width="150"  prop="ne_ip_address" ></el-table-column>
				<el-table-column label='<%=rb.getString("ShiJianLeiXing")%>' min-width="100" prop="event_name" ></el-table-column>
				<el-table-column label='<%=rb.getString("YuanYin")%>' min-width="200" prop="event_reason"></el-table-column>
				<el-table-column label='<%=rb.getString("ShiJian")%>' min-width="150" prop="time"></el-table-column>
			</el-ctable>
		</el-tab-pane>
	</el-tabs>
	
	<!-- 二级页面 -- 新建收集任务 -->
	<el-slide ref="slide" :url="slideUrl" :title="slideTitle" :footer="slideFooter" :header='slideHeader' :position="slidePosition"
	    :height="slideHeight" :modal='modal'  :width="slideWidth" :subloading="slideSubmitLoading" @ok='saveCollectLogTask' @cancel='cancelSlide' 
	    :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'" >
	 </el-slide>
	 
	 <!-- 二级页面 -- 查看收集任务  -->
	<el-slide ref="slideView" :title="slideTitleView" :footer="false" :header='slideHeader' position="bottom"
	    height="300px" :modal='modal'  :width="slideWidth" :subloading="slideSubmitLoading" @cancel='cancelViewSlide'>
	    	<el-ctable id="gnbLogDetail" ref="ctable" :time="6" :url="logDetailUrl" :height="height"
					:query-params="params_detail"   page-size="10" pagination="true" :rownumber=true>
				
				<!-- 主列表 -->
				<el-table-column label='<%=rb.getString("WenJianMing")%>' min-width="200"  prop="file_name" ></el-table-column>
				<el-table-column label='<%=rb.getString("ShouJiShiJian")%>' min-width="150" prop="upload_time" ></el-table-column>
				<el-table-column label='<%=rb.getString("CaoZuo")%>' min-width="120" width="120">
					<template slot-scope="scope">
	            		<div class="operationDiv el-icon el-icon-operation-download curpo" @click="downlodFile(scope.row,'File')" ></div>
	            		<div class="operationDiv el-icon el-icon-operation-delete CODE_GNB_LOGS hidden" @click="delCollectFile(scope.row,'File')"  ></div>
	          		</template>
				</el-table-column>
			</el-ctable>
	 </el-slide>
	 <form id = "exportEventLog" style="display:none" method="post"></form>
</div>
</div>
<script type="text/javascript">
var JieGuo = '<%=rb.getString("JieGuo")%>';
var gnbLogsVue = new Vue({
	el:'#gnbLogs',
	data:{
		activeName:'first',
		//设备上报日志
		params_report:{
			device_type:'eNB',
			device_code:'',
			search_text:'',			
			like_fields:'serial_number',
			timeZone:timeZone,		
			start_time:'',
			end_time:'',
            isGnb: 1,
		},
		//设备异常日志
		params_exception:{
			timeZone:timeZone,
			device_name:'',
			operate_type:'',
			op_start_time:'',
			op_end_time:'',
			search_text:'',
            isGnb: 1,
		},
		query_exception:{
			operate_type:'',
		},
		params_event:{
			timeZone:timeZone,
			id:'',
			neSerialNumber:'',
			eventName:'',
			neIpAddress:'',
			startTime:'',
			endTime:'',
			searchText:''
		},
		query_event:{
			eventName:'',
		},
		params_detail:{
			timeZone:timeZone,
			taskId:'',
            isGnb: 1
		},
		menus:[],
	    rowData:[],
	    selection:'',
	    slideUrl:'',
	    logDetailUrl:'',
	    slideTitle:'',
	    slideTitleView:'',
	    slideHeader:'',
	    slideHeaderView:'',
	    slideFooter:'',
	    slidePosition:'',
	    slidePositionView:'',
	    slideHeight:'',
	    slideHeightView:'',
	    slideWidth:'',
        slideSubmitLoading:'',
	    exceptionTypeOptions:[],
	    height:'100%',
	    width:'100%',
	    modal:false,
	    tabsShow: true,
	    statisticOptions:[],
		tableRule:{
			first:'ctableReport',
            second:'ctableAlarm',
            third:'ctableException',
            forth:'ctableEvent'
		},
		selectedList: [], //已选设备
		excTaskMap: {}, // 记录异常日志已选数据,
		bulkDeviceSelectShow: false, // 批量选择显示
		bulkExcSelectShow: false, // 异常日志批量选择显示
		queryExcDateTime: [], // 异常日志时间
		queryDeviceDateTime: [], // 设备日志时间
		queryEventDateTime: [], // 事件日志时间
		tabsCode: {
			'first': 'ctableReport',
			'second': 'ctableAlarm',
			'third': 'ctableException'
		},
        isAlreadyGlobalMaxNumOut:false,
	},
	computed: {
		selectedTitle(){
			return '<%=rb.getString("XiaoZhanBianMa")%>';
		},
		isWritable() {
			return writableMap['CODE_GNB_LOGS'] == true;
		},
	},
	mounted(){
		this.init();
		eventBus.$on('hide-collect',this.hideSlide);
		eventBus.$on('close-collect',this.cancelSlide);
	},
	watch:{
		// 监听筛选
		selection(){
			var activeName = this.$root.activeName;
			var data;
			if(activeName == 'first'){	    		
				data = this.selection;
	    	}else if(activeName == 'second'){
	    		data = this.$refs.ctableAlarm.getData();
	    	}else if(activeName == 'third'){
	    		data = this.$refs.ctableException.getData();
	    	}
			
			 
			var cellCodes = '';
			if(data.length != 0){
				data.map(function(item){
					cellCodes += item.serial_number + ","
				})
			}
			
		},
	},
	methods:{
		init(){    //根据权限判断页面默 认显示的tab内容  
			var vm = this;
		
			//获取异常类型信息 -- 填充下拉选项  
			axios.post('${ctx}/system/logmange/device/getOperationNameForCombobox.action',stringify({
				
			})).then(function(response){
				let data = response.data
				vm.exceptionTypeOptions = data;
			}).catch(function(error){})
			
			//获取事件日志类型 -- 填充下拉选项 
            /** 事件日志tab页暂时隐藏
			axios.post('${ctx}/cell/cellEventLog/getEventTypeForCombobox.action',stringify({
				
			})).then(function(response){
				let data = response.data
				vm.statisticOptions = data;
			}).catch(function(error){})
            */
		},
	   
		handerClose(){ //点击页面其他地方菜单收起
	        this.$refs.menuReport.hide();
	    },
		/**
		 * 更多操作
		 * @parame row:1.查看  2.终止  3.下载  4.删除 
		*/
		optClick(row,ev){ 
            var vm = this,
                activeName = vm.activeName,
	    	    status = row.task_status;  
	    	//0-收集未开始；1-正在收集；2-收集完成；3-收集失败；4-收集终止;5-等待上传； 
	    	
	    	//表格处理 0-等待， 1-进行中，2-成功，3-失败，4-终止，5-正在停止上报（进行中），6- 停止上报 成功（终止），7-停止上报失败（进行中），8-重启(等待)
			
			//5 正在停止上报  （进行中）：可终止，不可删除  (实际已在终止中，不可再点击终止操作）
	    	vm.rowData = row;
			var downloadFlag = false , terminateFlag = false , showFlag = false , delFlag = false, showDownlodFlag = false;
   
	    	if(status == 0 || status == 1 || status == 7 || status == 8){
	    		terminateFlag = true;
	    	}else{
	    		terminateFlag = false;
	    	}
	    		    			
	    	//进行中 - 表格操作-删除为置灰状态
	    	if(status == 1 || status == 5 || status == 7){
	    		delFlag = false;
	    	}else{
	    		delFlag = true;
	    	}
	    	
	    	if (row.file_num > 0 || activeName == 'third'){
	    		downloadFlag = true
	    	}else{
	    		downloadFlag = false
	    	}
	    	
	    	if(activeName == 'third'){
	    		showFlag = false;
	    	}else{
	    		showFlag = true;
	    	}
	    	//设备日志，表格去掉文件数量字段； 操作项去掉下载
	    	if(activeName == 'first'){
	    		showDownlodFlag = false;
	    	}else{
	    		showDownlodFlag = true;
	    	}
	    	this.menus= [
		          {label:'<%=rb.getString("JieGuo")%>',cls:"el-icon el-icon-operation-result",code:'view',show:showFlag},
		          {label:'<%=rb.getString("ZhongZhiRenWu")%>',cls:"el-icon el-icon-operation-terminate CODE_GNB_LOGS hidden" ,code:'terminate',disable:!terminateFlag ,show:showFlag},
		        //   {label:'<%=rb.getString("XiaZai")%>',cls:"el-icon el-icon-operation-download" ,code:'download',disable:!downloadFlag,show:showDownlodFlag},
		          {label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete CODE_GNB_LOGS hidden",code:'del',disable:!delFlag}
		    ]
	    	var vm = this;
	    	this.$nextTick(function(){
	    		document.body.click();

				if(activeName == 'first'){	    		
					vm.$refs.menuReport.show(ev);
		    	}
	    	});
	    },
		/**
		 *  获取点击项数据
		 * @parame ev:点击属性数据
		*/
	    clickMenu(ev){ //单点击方法 -- 设备上报日志、告警日志 
            var vm = this;
	    	var codes = {
	    		view:this.viewCollectTask,
	    		terminate:this.terminateCollectTask,
	    		download:this.downlodFile,
	    		del:this.delCollectFile
	    	}
	    	if(codes[ev.code]){
	    		codes[ev.code](vm.rowData,'Task')
	    	}
	    },
		/**
		 *  结果
		 * @param id:当前数据ID
		 * @param status:状体
		*/
	    viewCollectTask(row){   //查看任务进度  -- 设备上报日志、告警日志
			var vm = this,
                activeName = vm.activeName,
                id = vm.activeName == 'third'? row.id: row.task_id;

			vm.$refs.slideView.showSlide(function(){
	    		vm.modal = false;
	    	});
			vm.params_detail.taskId = id;
			if(activeName == 'first'){					
				var startTime = vm.rowData.start_time,
					endTime = vm.rowData.end_time,
					executeType = vm.rowData.execute_type;
				//根据执行方式:定时执行 标题后面增加 时间段
				if(executeType == 'Periodically'){
					vm.slideTitleView = JieGuo + '（' + startTime + ' 一   ' + endTime + '）';
				}else{
					vm.slideTitleView = JieGuo;
				} 
				
				vm.logDetailUrl = '${ctx}/cell/collect/getImmediateCollectLogFileDataList.action'
				
	    	}else if(activeName == 'second'){
	    		vm.slideTitleView = JieGuo;
	    		vm.logDetailUrl = '${ctx}/cell/collect/alarm/getCollectAlarmLogFileDataList.action'
	    	}
			
	    },
	    cancelViewSlide(){ // 二级页面关闭
	    	var vm = this;
	    	vm.$refs.slideView.hide();
	    },
		/**
		 *  //终止日志收集  -- 设备上报日志、告警日志
		 * @param id:当前数据id
		 * @param status:0-收集未开始；1-正在收集；2-收集完成；3-收集失败；4-收集终止;5-等待上传；
	    	//表格处理 0-等待， 1-进行中，2-成功，3-失败，4-终止，5-正在停止上报（进行中），6- 停止上报 成功（终止），7-停止上报失败（进行中），8-重启(等待)
		*/
	    terminateCollectTask(row){  
	    	var vm = this , 
                url = '',
                activeName = vm.activeName,
                status = row.task_status,
                id = vm.activeName == 'third'? row.id: row.task_id,
                params = {};
	    	if (status != 0 && status != 1 && status != 7 && status != 8) {
	    		showMsg('prompt_msg','<%=rb.getString("MeiYouKeTingZhiShouJiDeSheBei")%>');
	            return;
	        }

			if(activeName == 'first'){	    		
				url = '${ctx}/cell/collect/goTerminateImmediateCollectLogFile.action';
				params = {
					taskIds:id,
		    		device_code:vm.rowData.device_code,
		    		execute_type:vm.rowData.execute_type,
                    isGnb: 1
				}
				
	    	}else if(activeName == 'second'){
	    		url = '${ctx}/cell/collect/alarm/terminateTask.action';
	    		params = {
	    			taskIds:id,
                    isGnb: 1
				}
	    	}

	    	axios.post(url,stringify(params)).then(function(response){
	    		var data = response.data;
	    		if(data["success"]){
	    			vm.$refs.ctable.refresh()
	    			vm.$message({
		    			type:'success',
		    			message:'<%=rb.getString("ChengGong")%>'
		    		})
		    		vm.$refs[vm.tableRule[activeName]].refresh();
	    		}else{
	    			vm.$message.error(data["message"])
	    		}
	    	}).catch(function(error){
	    		
	    	})
	    },
		/**
		 *  下载文件
		 * @param row:当前数据
		*/
	    downlodFile(row,downType){ 
	    	var vm = this,
                fileNum = row.file_num ? row.file_num : 0,
                fileName = row.file_name ? row.file_name : '',
                activeName = vm.activeName,
                params = {},
                url = '',
	    		checkURL = '';
            
            vm.rowData = row;
	        //如果没有日志文件，提示没有文件
	        if (fileName == null && fileNum == 0) {
	       	 	showMsg('prompt_msg','<%=rb.getString("MeiYouYaoXiaZaiDeWenJian")%>');
	            return;
	        }  
	        if(activeName == 'first'){	 
				checkURL = '${ctx}/cell/collect/getDownloadFileNumber.action';   		
				url = "${ctx}/cell/collect/doDownloadImmediateCollectLogFile.action"
				params = {
					taskIds: row.task_id,
					timeZone: timeZone,
                    fileName:fileName,
                    isGnb: 1
				};
                if(downType == 'File'){
					params.fileName = fileName;
				}
	    	}else if(activeName == 'second'){
				checkURL = '${ctx}/cell/collect/alarm/getDownloadFileNumber.action';
	    		url = '${ctx}/cell/collect/alarm/downloadFile.action';
	    		params = {
                    taskIds: row.task_id,
                    fileName : '',
                    isGnb: 1
                };
                if(downType == 'File'){
					params.fileName = fileName;
				}
	    	}else if(activeName == 'third'){
				checkURL = '${ctx}/system/logmange/device/getExceptionLogNumber.action';
	    		url = '${ctx}/system/logmange/device/doDownloadExceptionLogFile.action';
	    		params = {
                    record_ids : row.id,
                    file_name : fileName,
                    isGnb: 1
	            };
	    	}
    		axios.post(checkURL,stringify(params)).then(function(response){
	    		var data = response.data;
	    		if(data.length > 0 || data.fileNum > 0){
	    			vm.createForm(url,params);
	    		}else{
	    			vm.$message.error('<%=rb.getString("WenJianBuCunZai")%>')
	    		}
	    	}).catch(function(error){
	    		
	    	})
   
	    },
		/**
		 * 删除任务  -- 设备上报日志 、告警日志 
		 * @param row:当前数据
		*/
	    delCollectFile(row,delType){
	    	var vm = this , 
                url='' ,
                activeName = vm.activeName;
                id = vm.activeName == 'third'? row.id: row.task_id,
	    		fileName = delType == 'Task' ? '' : row.file_name,
                params = {};
	    	
			if(activeName == 'first'){	    		
				url = '${ctx}/cell/collect/doClearImmediateCollectLogFile.action'
                params = {
                    taskIds:id,
                    fileName : fileName,
                    isGnb: 1
                };
	    	}else if(activeName == 'second'){
	    		url = '${ctx}/cell/collect/alarm/delTaskAndClearFile.action';
	    		params = {
                    taskIds: id,
                    fileName : fileName,
                    isGnb: 1
                };
	    	}else if(activeName == 'third'){
	    		url = '${ctx}/system/logmange/device/doClearExceptionLog.action';
	    		params = {
                    record_ids : id,
                    file_name : fileName,
                    isGnb: 1
	            };
	    	}
			
	    	vm.$confirm('<%=rb.getString("QueRenShanChuWenJian")%>',QueRen,{
	    		customClass:'warningConfirm',
	    		confirmButtonText:'<%=rb.getString("QueDing")%>',
	    		cancelButtonText:'<%=rb.getString("QuXiao")%>',
	    		type:'warning',
	    		closeOnClickModal:false
	    	}).then(() => {
	    		axios.post(url,stringify(params)).then(function(response){
		    		var data = response.data;
		    		if(data["success"]){
		    			vm.$refs.ctable.refresh()
		    			vm.$message({
			    			type:'success',
			    			message:'<%=rb.getString("ChengGong")%>'
			    		})
			    		vm.$refs[vm.tableRule[activeName]].refresh();
						vm.$refs.slideView.hide();
		    		}else{
		    			vm.$message.error(data["message"])
		    		}
                    vm.clearBulkSelected();
		    	}).catch(function(error){
		    		
		    	})
	    	}).catch()
	    },
	    statisEventLog(){    //跳转到统计功能页面  
			var vm = this;
	    	
			vm.$refs.slide.showSlide(function(){
	    		vm.modal = false;
	    	});
	    	vm.styleObj = {
	    		zIndex:10
	    	}
	    	vm.slideHeader = false;
	    	vm.slideFooter = false;
	    	vm.slideUrl = '${ctx}/cell/cellEventLog/goCellEventPage.action';
	    	vm.slidePosition = 'top';
	    	vm.slideHeight = '100%';
	    	vm.slideWidth = '100%';
            vm.slideSubmitLoading = false;
	   
			vm.$refs.slide.showSlide(function(){
	    		eventBus.$emit('statisticLog',vm.params_event)
	    		
	    	});
		},
		 //导出日志数据 	 --  事件日志
		exportLogs(){
			var vm = this;	

       		exportByForm('${ctx}/cell/cellEventLog/exportLogToCsvFile.action',{
       			timeZone: timeZone,
   				id: vm.params_event.id,
   				neSerialNumber: vm.params_event.neSerialNumber,
   				neIpAddress: vm.params_event.neIpAddress,
   				eventName: vm.params_event.eventName,
   				startTime: vm.params_event.startTime,
   				endTime: vm.params_event.endTime,
   				searchText: vm.params_event.searchText
       		});
		},
		 //导出日志数据-设备异常日志
		exceptionExportLogs(){
		    var vm = this;

		    exportByForm('${ctx}/system/logmange/device/exportExceptionLogToCsvFile.action',{
                timeZone: timeZone,
                device_name: vm.params_exception.device_name,
                operate_type: vm.params_exception.operate_type,
                op_start_time: vm.params_exception.op_start_time,
                op_end_time: vm.params_exception.op_end_time,
                search_text: vm.params_exception.search_text,
                isGnb: 1
            });
		},

		collectLogs(){  // 新建日志收集任务  
			var vm = this,
                activeName = vm.activeName;
			vm.cancelViewSlide();
			
	    	vm.styleObj = {
	    		zIndex:10
	    	}
	    	vm.slideHeader = true;
	    	vm.slideFooter = true;
	    	
	    	vm.slideUrl = '${ctx}/gnb/devicelogs/toGnodeBAddLogTaskPage.action';
	    	vm.slidePosition = 'top';
	    	vm.slideHeight = '100%';
	    	vm.slideWidth = '100%';
            vm.slideSubmitLoading = false;
			vm.$refs.slide.showSlide(function(){
				vm.modal = false;
			
	        	if(vm.activeName == 'first'){
	        		vm.slideTitle = '<%=rb.getString("XinJianSheBeiShangBaoRiZhiRenWu")%>';
	        	}else if(vm.activeName == 'second'){
	        		vm.slideTitle = '<%=rb.getString("XinJianGaoJingRiZhiRenWu")%>';
	        	}
	    	});
		},
		saveCollectLogTask(){ //  保存新建任务
			var vm = this,
                activeName = vm.activeName;
            if(activeName == 'first'){
                eventBus.$emit('collect-submit');
            }else{
                eventBus.$emit('hander-ok',activeName);
            }
	    },
	    cancelSlide(){// 退出新建任务页面
	    	var vm = this;
	    	vm.$refs.slide.hide();
	    	vm.styleObj = {
	    		zIndex:10
	    	}
	    	if(vm.activeName == 'first'){
	    		vm.$refs.ctableReport.refresh()
	    	}else if(vm.activeName == 'second'){
	    		vm.$refs.ctableAlarm.refresh()
	    	}
	    },
	    hideSlide(){// 关闭新建任务页面 
	    	var vm = this;
	    	vm.$refs.slide.hide();
			
	    	if(vm.activeName == 'first'){
	    		vm.$refs.ctableReport.refresh()
	    	}else if(vm.activeName == 'second'){
	    		vm.$refs.ctableAlarm.refresh()
	    	}
	    },
		/**
		 * 多选响应
		 * @param selection:选择的数据
		*/
	    batchSelect(selection){ 
	    	var vm = this,
	    		tbNames = {
					'first': 'ctableReport',
					'second': 'ctableAlarm',
					'third': 'ctableException'
				},
				tbName = tbNames[vm.activeName],
				tb = vm.$refs[tbName];
	    	vm.selection = selection
			
	    	if(tbName == 'ctableException' && selection){ // 存储历史日常日志数据
	    		selection.map(function(item){
	    			vm.excTaskMap[item.id] = item.serial_number + ' - ' + item.op_start_time;
	    		})
	    	}
	    	var cklist = tb.getChecked();
	    	
	    	setTimeout(function(){
		    	if(cklist && cklist.length) {
		    		vm.selectedList = cklist.map(function(value){
		    			var rowItem = {serial_number: value};
		    			
		    			if(tbName == 'ctableException'){
		    				rowItem.serial_number = vm.excTaskMap[value]
		    				rowItem.id = value;
		    			}
		    			return rowItem;
		    		})
		    	}else {
		    		vm.selectedList = [];
		    	}
	    	},10)
		},
		/**
		 * 清除当前tab的单个设备选择记录
		 * @param row:数据
		*/
		delSingleRecord(row) { 
			var vm = this,
				tbNames = {
					'first': 'ctableReport',
					'second': 'ctableAlarm',
					'third': 'ctableException'
				},
				tbName = tbNames[vm.activeName],
				rowkey = (vm.activeName == 'third')? 'id' : 'serial_number',
				tb = vm.$refs[tbName],
				rows = tb.getData(),
				cklist = tb.getChecked();
			rows.map(function(item){
				if(item[rowkey] == row[rowkey]) tb.toggleRowSelection(item, false);
			});
			
			cklist.splice(cklist.indexOf(row[rowkey]),1);
			
			vm.selectedList = vm.selectedList.filter(function(item){
				return item[rowkey] != row[rowkey];
			});
		},
		//批量下载任务下收集的所有日志文件
		downloadImmediateLogFile() {
			var vm = this,
				tbNames = {
					'first': 'ctableReport',
					'second': 'ctableAlarm',
					'third': 'ctableException'
				},
				tbName = tbNames[vm.activeName],
				rowKey = (vm.activeName == 'third')? 'id' : 'serial_number',
				tb = vm.$refs[tbName],
				logFileSelect = tb.getChecked(),
				tbData = tb.getData(),
				files = [],
				checkURL = '',
				downURL = '',
				fileName = '',
				params = {timeZone: timeZone};
			
			tbData.map(function(row){
				var sn = row[rowKey],
					fileNum = row.file_num,
					taskId = row.task_id;
				
				if(vm.activeName == 'third') taskId = row.id;
				fileName = row.file_name || '';
				if(logFileSelect.includes(sn)) {
					if(fileNum>0 || fileNum == undefined) files.push(taskId);
				}
			});
			
			if(vm.selection.length <= 0)return

			if(vm.activeName == 'first') {// 设备日志
				checkURL = '${ctx}/cell/collect/getDownloadFileNumber.action';
				downURL = '${ctx}/cell/collect/doDownloadImmediateCollectLogFile.action';
				params.taskIds = files.join(',');
				params.fileName = fileName;
			}
			if(vm.activeName == 'second') {// 告警日志
				checkURL = '${ctx}/cell/collect/alarm/getDownloadFileNumber.action';
				downURL = '${ctx}/cell/collect/alarm/downloadFile.action';
				params.taskIds = files.join(',');
				params.fileName = fileName;
			}
			if(vm.activeName == 'third') {// 设备异常日志
				checkURL = '${ctx}/system/logmange/device/getExceptionLogNumber.action';
				downURL = '${ctx}/system/logmange/device/doDownloadExceptionLogFile.action';
				params.record_ids = files.join(',');
				params.file_name = fileName;
			}
			
			if(files.length == 0) {
				showMsg('prompt_msg','<%=rb.getString("MeiYouYaoXiaZaiDeWenJian")%>');
			    return;
			}
			
			$.post(checkURL, params, function (data) {
		        if (data.length > 0 || data.fileNum > 0) {
		    		vm.createForm(downURL,params);
		        } else {
		       	 	showMsg('prompt_msg','<%=rb.getString("WenJianBuCunZai")%>');
		            return;
		        }
		    }, "json");
		},
		//批量删除任务收集日志文件
		batchDeleteForLog(){
			var vm = this,
				tbNames = {
					'first': 'ctableReport',
					'second': 'ctableAlarm',
					'third': 'ctableException'
				},
				tbName = tbNames[vm.activeName],
				rowKey = (vm.activeName == 'third')? 'id' : 'serial_number',
				tb = vm.$refs[tbName],
				logFileSelect = tb.getChecked(),
				tbData = tb.getData(),
				files = [], curInProgress = [];
			
			tbData.map(function(row){
				var sn = row[rowKey],
					fileNum = row.file_num,
					taskId = row.task_id;
				if(vm.activeName == 'third') taskId = row.id;
				if(logFileSelect.includes(sn)) {
					files.push(taskId);
				}
			});

            if(vm.selection.length <= 0)return
            
			vm.selection.map(function(row){
				if(row.task_status =='1' || row.task_status =='5' || row.task_status =='7'){
					curInProgress.push(row.task_status);
				}
			});
			if(curInProgress.length > 0 ){
                vm.$confirm('<%=rb.getString("JinXingZhongDeRenWuBuKeYiShanChu")%>','<%=rb.getString("QueRen")%>',{
                    customClass:"warningConfirm",
                    confirmButtonText:'<%=rb.getString("QueDing")%>',
                    cancelButtonText:'<%=rb.getString("QuXiao")%>',
                    type:'warning',
                    closeOnClickModal:false,
                    dangerouslyUseHTMLString:true
                }).then(() => {
                    vm.clearBulkSelected();
                })
				return;
			}else{
                vm.$confirm('<%=rb.getString("QueRenShanChuRenWuJiLuHeWenJian")%>','<%=rb.getString("QueRen")%>',{
                    customClass:"warningConfirm",
                    confirmButtonText:'<%=rb.getString("QueDing")%>',
                    cancelButtonText:'<%=rb.getString("QuXiao")%>',
                    type:'warning',
                    closeOnClickModal:false,
                    dangerouslyUseHTMLString:true
                }).then(() => {
                    var params = {
                            "taskIds": files.join(',')
                        },
                        url = '${ctx}/cell/collect/doClearImmediateCollectLogFile.action';

                    if(vm.activeName == 'third'){// 设备异常日志
                        params = {
                            "record_ids": files.join(',')
                        };
                        url = '${ctx}/system/logmange/device/doClearExceptionLog.action';
                    }
                    
                    if(vm.activeName == 'second'){
                        url = '${ctx}/cell/collect/alarm/delTaskAndClearFile.action';
                    }
                    
                    axios.post(url,stringify(params)).then(function(response){
                        tb.refresh();
                        showMsg('success_msg','<%=rb.getString("ChengGong")%>');
                        vm.clearBulkSelected();
                    }).catch(function(error){})
                })
			}
		},
		/**
		 * 跳转from表格
		 * @param url:地址
		 * @param params：跳转参数
		*/
		createForm(url,param) {
			var body = document.querySelector('body'),
				form = document.createElement('form'),
				params = param || {};
			
			form.style.display = 'none';
			form.action = url;
			form.method = 'post';
			
			if(params) {
				params.token = omctoken;
				for(var key in params) {
					var input = document.createElement('input');
					input.value = params[key];
					input.setAttribute('name',key);
					form.appendChild(input);
				}
			}
			
			body.appendChild(form);
			form.submit();
			form.remove();
			this.clearBulkSelected();
		},
		selectChange(selection){ // 全选操作
			this.selection = selection // 获取选择的数据
		},
		//设备上报日志-模糊查询
		queryDevice:function(val){
			var vm = this;
			vm.params_report.search_text  = val;
		},
		//异常日志 模糊查询
		queryExc:function(val){   
			var vm = this;
			vm.params_exception.search_text = val;
		},
		queryEvent:function(val){
			var vm = this;
			this.params_event.searchText  = val;
		},
	    tabClick(tab){ // 列表的点击
	    	var vm = this;

	    	vm.cancelViewSlide();
	    	vm.selection = '';
	    	
			var activeName = this.$root.activeName
			if(activeName != 'forth') vm.batchSelect('');
	    },
		collectLogFile(row){
			var vm = this,
				url = '${ctx}/system/logmange/device/manualSetCellErrorLog.action',
				params = {
					timeZone: timeZone,
					smallCellCode:row.device_code,
					id:row.id
				};
              
            var confirmHint ='<div style="font-size:14px;font-weight: 550;">'+ '<%=rb.getString("ShouDongShouJiYiChangRiZhiTiShiOne")%>' +'</div>'+'<div style="font-size:14px;">'+ '<%=rb.getString("ShouDongShouJiYiChangRiZhiTiShiTwo")%>' +'</div>';
            if(vm.isAlreadyGlobalMaxNumOut){
                vm.$confirm(confirmHint,'<%=rb.getString("QueRen")%>',{
                    customClass:"warningConfirm",
                    confirmButtonText:'<%=rb.getString("QueDing")%>',
                    cancelButtonText:'<%=rb.getString("QuXiao")%>',
                    type:'warning',
                    closeOnClickModal:false,
                    dangerouslyUseHTMLString:true
                }).then(() => {
                    axios.post(url,stringify(params)).then(function(response){
                        var data = response.data;
                        if(data["success"]){
                            vm.$message({
                                message:'<%=rb.getString("ChengGong")%>',
                                type:'success',
                            })
                            vm.$refs.ctableException.refresh();
                        }else{
                            vm.$message.error(data["message"])
                        }
                    }).catch(() => {})
                })
            } else{
                axios.post(url,stringify(params)).then(function(response){
                    var data = response.data;
                    if(data["success"]){
                        vm.$message({
                            message:'<%=rb.getString("ChengGong")%>',
                            type:'success',
                        })
                        vm.$refs.ctableException.refresh();
                    }else{
                        vm.$message.error(data["message"])
                    }
                }).catch(() => {})
            }
			
		},
		// 打开已选弹窗
		openBulkSelectTable(){
			var vm = this;
			if(vm.activeName == 'first'){
				vm.bulkDeviceSelectShow = true;
			}else if(vm.activeName == 'third'){
				vm.bulkExcSelectShow = true;
			}
		},
		// 关闭已选弹窗
		closeBulkSelectTable(){
			var vm = this;
			if(vm.activeName == 'first'){
				vm.bulkDeviceSelectShow = false;
			}else if(vm.activeName == 'third'){
				vm.bulkExcSelectShow = false;
			}
		},
		// 设备已选表格 清空事件
		clearBulkSelected(){
			var vm = this;
			
			vm.$refs[vm.tabsCode[vm.activeName]].clearSelection();
			if(vm.activeName == 'first'){
				vm.bulkDeviceSelectShow = false;
			}else if(vm.activeName == 'third'){
				vm.bulkExcSelectShow = false;
			}
		},
		// 设备已选表格 单个删除事件
		delBulkSelected(rows){
			var vm = this,
				tabs = vm.tabsCode[vm.activeName],
				rowKey = (vm.activeName == 'third')? 'id' : 'serial_number';
			vm.selectedList = vm.selectedList.filter((items)=>{
				return items[rowKey] != rows[rowKey]
			});
			var selection = this.$refs[tabs].$refs.ctableInner.store.states.selection,
				irow= selection.filter((items)=>{
					return items[rowKey] == rows[rowKey]
				})[0];
			vm.$refs[tabs].toggleRowSelection(irow,false);
			var idx = vm.$refs[tabs].ckList.indexOf(rows[rowKey]);
			vm.$refs[tabs].ckList.splice(idx,1);
			
			if(vm.selectedList.length == 0){
				if(vm.activeName == 'first'){
					vm.bulkDeviceSelectShow = false;
				}else if(vm.activeName == 'third'){
					vm.bulkExcSelectShow = false;
				}
			}
		},
		// 设备日志时间控件查询
		queryDeviceDateTimeChange(dateTime) {
			var vm = this;
			vm.queryDeviceDateTime = dateTime;
			if(dateTime != null){
				vm.params_report.start_time = dateTime[0];
				vm.params_report.end_time = dateTime[1];
			}else{
				vm.params_report.start_time = '';
				vm.params_report.end_time = '';
			} 
		},
		// 异常日志时间控件查询
		queryExcDateTimeChange(dateTime) {
			var vm = this;
			vm.queryExcDateTime = dateTime;
			if(dateTime != null){
				vm.params_exception.op_start_time = dateTime[0];
				vm.params_exception.op_end_time = dateTime[1];
			}else{
				vm.params_exception.op_start_time = '';
				vm.params_exception.op_end_time = '';
			}
		},
		// 事件日志时间控件查询
		queryEventDateTimeChange(dateTime) {
			var vm = this;
			vm.queryEventDateTime = dateTime;
			if(dateTime != null){
				vm.params_event.startTime = dateTime[0];
				vm.params_event.endTime = dateTime[1];
			}else{
				vm.params_event.startTime = '';
				vm.params_event.endTime = '';
			}
		},
		// 高级查询 确定事件
		advanceQuery(){
			var vm = this,
			type = vm.activeName;
			if(type == 'third'){
				Object.assign(vm.params_exception, vm.query_exception);
			}else if(type == 'forth'){
				Object.assign(vm.params_event, vm.query_event);
			}
		},
        // 异常日志表格请求成功回调
        loadSuccessExcTable(data){
	    	this.isAlreadyGlobalMaxNumOut = (data.properties && data.properties.isAlreadyGlobalMaxNumOut == true) ? true : false;
	    },
        // 进行中的不可选
		checkSelectTable(row,index){
			return row.manual_collection_status != '1';
		},
	}
})
</script>