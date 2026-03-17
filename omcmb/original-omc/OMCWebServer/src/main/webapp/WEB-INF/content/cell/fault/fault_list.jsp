<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style>
.alarmMinor , .alarmMajor , .alarmCritical , .alarmWarning{
	position:absolute;
	top:0;
	bottom:0;
	left:0;
	right:0;
	line-height:34px;
	padding-left:10px;
}
.alarmMinor{
	background:#CCCC66;
}
.alarmMajor{
	background:#DCAA5E;
}
.alarmCritical{
	background:#E88282;
}
.alarmWarning{
	background:#60BEFC;
}
.queryInfo{
	display:inline-block;
	margin-right:35px;
}
.queryInfo label{
	display:block;
	line-height:24px;
}
.tipHeader{
	margin-bottom:20px;
	color:#5a7b92;
	font-size:14px;
	font-weight:bold;
}
.tipHeader .titleIcon_close{
	display:inline-block;
	float:right;
	width:20px;
	height:20px;
	cursor:pointer;
}
span[class*='status_']{
	padding-left:25px;
}
.bottom-slider {
	position: absolute;
	bottom: -110px;
	width: calc(100% - 30px);
	padding: 12px 15px;
	background: #fff;
	box-shadow: 10px 0 50px rgba(0,0,0,0.15);
	transition: bottom .5s ease;
	z-index: 98;
}
.bottom-slider.show {
	bottom: 0px;
}
.selected-alarm-info {
	padding: 15px 15px 15px 0;
	height: 400px;
	width: 360px;
	position: absolute;
	bottom: -450px;
	background: #fff;
	box-shadow: 10px 0 50px rgba(158,200,222,0.45);
	transition: bottom .5s ease;
	z-index: 90;
}
.selected-alarm-info.show {
	bottom: 60px;
}
.list-title {
	font-size: 16px;
	color: #5A7B92;
	padding: 10px 0 10px 25px;
}
.list-body {
	width: calc(100% - 45px);
	padding-left: 25px;
	overflow: auto;
	height: 345px;
}
.list-body-title {
	background-color: #FBFBFB;
	color:#5A7B92;
	font-size: 12px;
	font-weight: bold;
	padding: 5px;
	border-bottom: 1px dashed silver;
}
.list-item-info {
	position: relative;
	padding: 6px 10px;
	border-bottom: 1px dashed silver;
	height:31px;
	box-sizing:border-box;
}
.list-item-op {
	padding: 5px;
	display: inline-block;
	position: absolute;
	top: 0px;
	right: 5px;
	color: red;
	cursor: pointer
}
.el-table .cell.el-tooltip{
	min-width: 0px !important
}
</style>

<div id="falutListPage" style="height:100%;background:#FFFFFF;">
	<!-- 按钮  -- 导出 -->
	<el-popover placement="bottom" trigger="manual" v-model="exportFlag">
		<div style="width:500px;height:450px;position:relative">
			<div class="cardHeader">
				<span><%=rb.getString("SheBeiZu")%></span> <span class="slideIcon el-icon el-icon-close" @click="closeExport"></span>
			</div>
			<el-ctable ref="ctableDeviceGroup" height="350px" url="${ctx}/cell/fault/getDeviceGroup.action" @selection-change="deviceGroupSelect"
				pagination="true" :rownumber=true row-key="groupId">
				<!-- 主列表 -->
				<el-table-column type="selection"></el-table-column>
				<el-table-column label='<%=rb.getString("SheBeiZuMingCheng")%>'  prop="groupName" ></el-table-column>
			</el-ctable>
			<p style='color:#FA5555;margin-bottom:5px;' v-show="showDeviceMsg"><%=rb.getString("QingXuanZeSheBeiZu")%></p>
			<span slot="footer" style="position:absolute;left:10px;bottom:10px;">
				<el-button type="primary" @click="exportAlarm"><%=rb.getString("QueDing")%></el-button>
				<el-button @click="closeExport"><%=rb.getString("QuXiao")%></el-button>
			</span>	
		</div>
		<div slot="reference" class="circleIcon placeholder-bt" placeholder="<%=rb.getString("DaoChu")%>">		
			<span class="el-icon-circle-export el-icon" @click="openExportDialg"></span>
		</div>
	</el-popover>
	
	<el-tabs v-model="faultType" @tab-click="handleClick" style="height:100%;">
		<el-tab-pane name="activeFault" label='<%=rb.getString("HuoDongGaoJing")%>' style="overflow: auto;"> <%-- url="${ctx}/cell/fault/queryFaultPageList.action"     :data="alarmData"  --%>
			<el-ctable id="activeFaultTable" ref="ctableActiveFault" :height="height"  url="${ctx}/cell/fault/queryFaultPageList.action"   @selection-change='alarmSelect'
				 :query-params="params_activeFault" :page-list="pageList" :limit="100" pagination="true" :rownumber=true @sort-change="sortChangeActive" :row-key="'alarm_id'">
				<template slot="toolbar">
					<el-query @query="activeAlarmQuery" @advance-query="activeAlarmQueryAdvance" @reset="activeAlarmReset" ok-text="<%=rb.getString("ChaXun")%>" reset-text="<%=rb.getString("ChaXunChongZhi")%>"
						placeholder="<%=rb.getString("GaoJingWeiYiBiaoZhi")%>/<%=rb.getString("KeNengYuanYin")%>/<%=rb.getString("WangYuanDingWei")%>">
						<template slot="form">
							<div class='queryInfo'>
								<label><%=rb.getString("YanZhongChengDu")%></label>
								<el-select filterable v-model="params_active.SERVERITY_IDS">
									<el-option v-for="item in alarmLevel" :label="item.text" :value="item.id"></el-option>
								</el-select>
							</div>
							<div class='queryInfo'>
								<label><%=rb.getString("GaoJingWeiYiBiaoZhi")%></label>
								<el-input v-model="params_active.ALARM_IDENTIFIER" size="mini" style='width:200px'></el-input>
							</div>
							<div class='queryInfo'>
								<label><%=rb.getString("ShiJianLeiXing")%></label>
								<el-select filterable v-model="params_active.EVENT_TYPE">
									<el-option v-for="item in eventType" :label="item.text" :value="item.id"></el-option>
								</el-select>
							</div>
							<div class='queryInfo'>
								<label><%=rb.getString("KeNengYuanYin")%></label>
								<el-input v-model="params_active.PROBABLE_CAUSE" size="mini" style='width:200px'></el-input>
							</div>
							<div class='queryInfo'>
								<label><%=rb.getString("SheBeiLeiXing")%></label>
								<el-select filterable v-model="params_active.NE_TYPE">
									<el-option v-for="item in deviceType" :label="item.text" :value="item.value"></el-option>
								</el-select>
							</div>
							<div class='queryInfo'>
								<label><%=rb.getString("WangYuanDingWei")%></label>
								<el-input v-model="params_active.EQUIP_INFO" size="mini" style='width:200px'></el-input>
							</div>
							<div class='queryInfo'>
								<label><%=rb.getString("GaoJingZhuangTai")%></label>
								<el-select filterable v-model="params_active.DEAL_STATE">
									<el-option v-for="item in alarmStatusActive" :label="item.text" :value="item.id"></el-option>
								</el-select>
							</div>
							<div class='queryInfo'>
								<label><%=rb.getString("GuZhangShiJian")%></label>
								<el-date-picker size="mini" v-model="params_active.time"  value-format="yyyy-MM-dd HH:mm:ss" type="datetimerange" range-separator="——"  start-placeholder='<%=rb.getString("KaiShiShiJian")%>' end-placeholder='<%=rb.getString("JieShuShiJian")%>'></el-date-picker>
							</div>
						</template>
					</el-query>
				</template>
				<!-- 主列表 -->
				<el-table-column type="selection" :reserve-selection="true" v-if="selectionShow"></el-table-column>
				<el-table-column label='' width="30" prop="">
					<template slot-scope="scope">
	            		<div class="el-icon el-icon-operation-more" @click="optClick(scope.row,event)" v-clickoutside="handerClose" style="cursor: pointer;"></div>
	          		</template>
				</el-table-column>
				<el-table-column label='<%=rb.getString("XuHao")%>' min-width="80" prop="alarm_id" sortable></el-table-column>
				<el-table-column label='<%=rb.getString("GaoJingJiBie")%>' min-width="100"  prop="alarm_serverity_value" sortable>
					<template slot-scope="scope">
						<div v-if="scope.row.alarm_serverity_value == 'Minor'">
							<span class="alarmMinor"><%=rb.getString("CiYaoGaoJing")%></span>
						</div>
						<div v-else-if="scope.row.alarm_serverity_value == 'Major'">
							<span class="alarmMajor"><%=rb.getString("ZhuYaoGaoJing")%></span>
						</div>
						<div v-else-if="scope.row.alarm_serverity_value == 'Critical'">
							<span class="alarmCritical"><%=rb.getString("JinJiGaoJing")%></span>
						</div>
						<div v-else-if="scope.row.alarm_serverity_value == 'Warning'">
							<span class="alarmWarning"><%=rb.getString("JingGaoGaoJing")%></span>
						</div>
	          		</template>
				</el-table-column>
				<el-table-column label='<%=rb.getString("GaoJingWeiYiBiaoZhi")%>' min-width="130" prop="alarm_identifier" sortable></el-table-column>
				<el-table-column label='<%=rb.getString("KeNengYuanYin")%>' min-width="180" prop="alarm_name" show-overflow-tooltip></el-table-column>
				<el-table-column label='<%=rb.getString("SheBeiLeiXing")%>' min-width="100" prop="ne_type"></el-table-column>
				<el-table-column label='<%=rb.getString("WangYuanDingWei")%>' min-width="250"  prop="equip_info" show-overflow-tooltip></el-table-column>
				<el-table-column label='<%=rb.getString("ShiJianLeiXing")%>' min-width="150" prop="event_type_value" show-overflow-tooltip></el-table-column>
				<el-table-column label='<%=rb.getString("GaoJingZhuangTai")%>' min-width="180" prop="deal_state" sortable show-overflow-tooltip>
					<template slot-scope="scope">
						<div v-if="scope.row.deal_state == '0'">
							<span class="status_unconfirmInactive"><%=rb.getString("WeiQueRenWeiQingChu")%></span>
						</div>
						<div v-else-if="scope.row.deal_state == '1'">
							<span class="status_confirmInactive"><%=rb.getString("YiQueRenWeiQingChu")%></span>
						</div>
	          		</template>
				</el-table-column>
				<el-table-column label='<%=rb.getString("GuZhangShiJian")%>' min-width="150" prop="event_time" sortable></el-table-column>
				<el-table-column label='<%=rb.getString("GengXinShiJian")%>' min-width="150"  prop="upd_time" sortable></el-table-column>
				<el-table-column label='<%=rb.getString("GaoJingFaShengCiShu")%>' min-width="100" prop="alarm_count" sortable></el-table-column>

			</el-ctable>
			<el-cmenu ref="menuActiveFault" :data="menus" @click="clickMenu"></el-cmenu>
		</el-tab-pane>
		<el-tab-pane name="historyFault" label='<%=rb.getString("LiShiGaoJing")%>' style="overflow: auto;"><%-- url="${ctx}/cell/fault/queryFaultHisPageList.action" :data="alarmHisData" --%>
			<el-ctable id="historyFaultTable" ref="ctableHistoryFault" :height="height" url="${ctx}/cell/fault/queryFaultHisPageList.action" @selection-change='alarmSelect'
				 :query-params="params_historyFault" :page-list="pageList" :limit="100" pagination="true" :rownumber=true @sort-change="sortChangeHistory" :row-key="'alarm_id'">
				<template slot="toolbar">
					<el-query @query="historyAlarmQuery" @advance-query="historyAlarmQueryAdvance" @reset="historyAlarmReset" ok-text="<%=rb.getString("ChaXun")%>" reset-text="<%=rb.getString("ChaXunChongZhi")%>"
						placeholder="<%=rb.getString("GaoJingWeiYiBiaoZhi")%>/<%=rb.getString("KeNengYuanYin")%>/<%=rb.getString("WangYuanDingWei")%>">
						<template slot="form">
							<div class='queryInfo'>
								<label><%=rb.getString("YanZhongChengDu")%></label>
								<el-select filterable v-model="params_history.SERVERITY_IDS">
									<el-option v-for="item in alarmLevel" :label="item.text" :value="item.id"></el-option>
								</el-select>
							</div>
							<div class='queryInfo'>
								<label><%=rb.getString("GaoJingWeiYiBiaoZhi")%></label>
								<el-input v-model="params_history.ALARM_IDENTIFIER" size="mini" style='width:200px'></el-input>
							</div>
							<div class='queryInfo'>
								<label><%=rb.getString("ShiJianLeiXing")%></label>
								<el-select filterable v-model="params_history.EVENT_TYPE">
									<el-option v-for="item in eventType" :label="item.text" :value="item.id"></el-option>
								</el-select>
							</div>
							<div class='queryInfo'>
								<label><%=rb.getString("KeNengYuanYin")%></label>
								<el-input v-model="params_history.PROBABLE_CAUSE" size="mini" style='width:200px'></el-input>
							</div>
							<div class='queryInfo'>
								<label><%=rb.getString("SheBeiLeiXing")%></label>
								<el-select filterable v-model="params_history.NE_TYPE">
									<el-option v-for="item in deviceType" :label="item.text" :value="item.value"></el-option>
								</el-select>
							</div>
							<div class='queryInfo'>
								<label><%=rb.getString("WangYuanDingWei")%></label>
								<el-input v-model="params_history.EQUIP_INFO" size="mini" style='width:200px'></el-input>
							</div>
							<div class='queryInfo'>
								<label><%=rb.getString("GaoJingZhuangTai")%></label>
								<el-select filterable v-model="params_history.DEAL_STATE">
									<el-option v-for="item in alarmStatusHistory" :label="item.text" :value="item.id"></el-option>
								</el-select>
							</div>
							<div class='queryInfo'>
								<label><%=rb.getString("GuZhangShiJian")%></label>
								<el-date-picker size="mini" v-model="params_history.time"  value-format="yyyy-MM-dd HH:mm:ss" type="datetimerange" range-separator="——"  start-placeholder='<%=rb.getString("KaiShiShiJian")%>' end-placeholder='<%=rb.getString("JieShuShiJian")%>'></el-date-picker>
							</div>
						</template>
					</el-query>
				</template>
				<!-- 主列表 -->
				<el-table-column type="selection"  :reserve-selection="true" v-if="selectionShow" ></el-table-column>
				<el-table-column label='' width="30" prop="">
					<template slot-scope="scope">
	            		<div class="el-icon el-icon-operation-more" @click="optClick(scope.row,event)" v-clickoutside="handerClose" style="cursor: pointer;"></div>
	          		</template>
				</el-table-column>
				<el-table-column label='<%=rb.getString("XuHao")%>' min-width="80" prop="alarm_id" sortable></el-table-column>
				<el-table-column label='<%=rb.getString("GaoJingJiBie")%>' min-width="100"  prop="alarm_serverity_value" sortable>
					<template slot-scope="scope">
						<div v-if="scope.row.alarm_serverity_value == 'Minor'">
							<span class="alarmMinor"><%=rb.getString("CiYaoGaoJing")%></span>
						</div>
						<div v-else-if="scope.row.alarm_serverity_value == 'Major'">
							<span class="alarmMajor"><%=rb.getString("ZhuYaoGaoJing")%></span>
						</div>
						<div v-else-if="scope.row.alarm_serverity_value == 'Critical'">
							<span class="alarmCritical"><%=rb.getString("JinJiGaoJing")%></span>
						</div>
						<div v-else-if="scope.row.alarm_serverity_value == 'Warning'">
							<span class="alarmWarning"><%=rb.getString("JingGaoGaoJing")%></span>
						</div>
					</template>
				</el-table-column>
				<el-table-column label='<%=rb.getString("GaoJingWeiYiBiaoZhi")%>' min-width="130" prop="alarm_identifier" sortable></el-table-column>
				<el-table-column label='<%=rb.getString("KeNengYuanYin")%>' min-width="180" prop="alarm_name" show-overflow-tooltip></el-table-column>
				<el-table-column label='<%=rb.getString("SheBeiLeiXing")%>' min-width="100" prop="ne_type"></el-table-column>
				<el-table-column label='<%=rb.getString("WangYuanDingWei")%>' min-width="250"  prop="equip_info" show-overflow-tooltip></el-table-column>
				<el-table-column label='<%=rb.getString("ShiJianLeiXing")%>' min-width="150" prop="event_type_value" show-overflow-tooltip></el-table-column>
				<el-table-column label='<%=rb.getString("GaoJingZhuangTai")%>' min-width="180" prop="deal_state" sortable show-overflow-tooltip>
					<template slot-scope="scope">
						<div v-if="scope.row.deal_state == '2'">
							<span class="status_unconfirmActive"><%=rb.getString("WeiQueRenYiQingChu")%></span>
						</div>
						<div v-else-if="scope.row.deal_state == '3'">
							<span class="status_confirmActive"><%=rb.getString("YiQueRenYiQingChu")%></span>
						</div>
	          		</template>
				</el-table-column>
				<el-table-column label='<%=rb.getString("GuZhangShiJian")%>' min-width="150" prop="event_time" sortable></el-table-column>
				<el-table-column label='<%=rb.getString("GaoJingQingChuShiJian")%>' min-width="150"  prop="clear_time" sortable></el-table-column>
				<el-table-column label='<%=rb.getString("GaoJingFaShengCiShu")%>' min-width="100" prop="alarm_count" sortable></el-table-column>

			</el-ctable>
			<el-cmenu ref="menuHistoryFault" :data="menus" @click="clickMenu"></el-cmenu>
		</el-tab-pane>
	</el-tabs>
	
	<el-slide ref="slide" :url="detailUrl" :title="'<%=rb.getString("XiangQing")%>'" :footer="false" :header='true' position="right" 
			:height="height" :modal='modal'  :width="width" @cancel="closeDetailFault">
	</el-slide>
	
	<el-dialog :title='dialogTitle' :visible.sync="showConfirmInfo" width="500" 
		:close-on-click-modal="false" :style="styleObj" @close="dialogClose">
		<el-form v-if="!clearFlag" :model="confirmForm" ref="confirmForm" label-position="top">
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
		<el-form v-if="clearFlag" :model="confirmForm" ref="confirmForm" label-position="top">
			<el-form-item v-if="!operationType">
				<%=rb.getString("QueRenQingChuGaoJing")%>
			</el-form-item>
			<el-form-item v-if="operationType">
				<%=rb.getString("QueRenPiLiangQingChuGaoJing")%>
			</el-form-item>
			<el-form-item label="<%=rb.getString("MiaoShu")%>" prop="description"> <!-- v-if="!confirmFlag" -->
				<el-input type="textarea" :rows="3" v-model="confirmForm.description">
			</el-form-item>
		</el-form>
		<span slot="footer" class="dialog-footer">
			<div class="buttonGroup">
				<el-button type="primary" @click="confirmAlarm"><%=rb.getString("QueDing")%></el-button>
				<el-button @click="closeConfirmInfo"><%=rb.getString("QuXiao")%></el-button>
			</div>	
		</span>
	</el-dialog>
	<!-- 批量操作 -->
	<div :class="bottomSliderCls">
		<div style="display: flex;">
			<div style="flex: auto;padding: 10px;display: flex;">
				<%=rb.getString("YiXuanGaoJing")%>（<span id="device_count" style="color: blue;">{{selectedList.length}}</span>）
				<span id="sliderArrow" class='el-icon' :class="{'el-icon-circle-down': !selectedAlarmShow, 'el-icon-circle-up':selectedAlarmShow}" style="font-size:16px; cursor: pointer;" @click="selectedAlarmShow = !selectedAlarmShow"> </span>
			</div>
			<div style="padding-top:8px;">
				<div class="linkbuttonGroup">
					<a class="linkbutton linkbutton_trend CODE_ALARM_VIEW hidden" @click="confirmBatch">
						<span><%=rb.getString("QueRenGaoJing")%></span>
					</a>
					<a class="linkbutton linkbutton_trend CODE_ALARM_VIEW hidden" @click="unConfirmBatch">
						<span><%=rb.getString("FanQueRenGaoJing")%></span>
					</a>
					<a class="linkbutton linkbutton_trend CODE_ALARM_VIEW hidden" @click="clearBatch" v-if="faultType == 'activeFault' ">
						<span><%=rb.getString("QingChuGaoJing")%></span>
					</a>
					<a class="linkbutton linkbutton_trend CODE_ALARM_VIEW hidden"  @click="deleteBatch" v-if="faultType == 'historyFault' ">
						<span><%=rb.getString("ShanChuGaoJing")%></span>
					</a>
					<a class="linkbutton linkbutton_nowanna" @click="cancelBatchOpt">
						<span><%=rb.getString("QuXiao")%></span>
					</a>
				</div>
			</div>
		</div>
	</div>
	<!-- 查看已选告警框 -->
	<div :class="selectedAlarmInfoCls">
		<div class="enb-list-ctn">
			<div class="list-title">
				<%=rb.getString("YiXuanGaoJing")%>
				<span style="float: right;padding-right: 15px;" @click="selectedAlarmShow=false"><i class="el-icon el-icon-close"></i></span>
			</div>
			<div style="width: calc(100% - 44px);padding-left: 25px;">
				<div class="list-body-title">
					{{selectedTitle}}
					<div class=" operation_delete" title="Delete All" style="float: right;padding-left: 50px;width: 30px;" @click="delAllSelectedRecord"><%=rb.getString("QingKong")%></div>
				</div>
			</div>
			<div class="list-body" style="flex: auto;">
				<div v-for="(item,index) in selectedList" :key="index" class="list-item-info">{{item.alarm_id.alarm_id}}({{item.alarm_id.serial_number || '.'}})<span class="list-item-op" style='font-size:16px;' @click="delSingleRecord(item)">x</span></div>
			</div>
		</div>
	</div>
</div>

<script>
	var faultListVue = new Vue({
		el:'#falutListPage',
		data(){
			return{
				faultType:'activeFault',
				height:'100%',
				pageSize:100,
				pageList:[30,50,100],
				menus:[], // 表格点击列表
				detailUrl:'',
				modal:true,
				width:'800px',
				timeZone:timeZone,
				alarmData:[], // 活动告警列表数据
				alarmHisData:[], // 历史告警列表数据
				params_activeFault:{  //活动告警  -- 查询参数 
					timeZone : timeZone,
					like_fields : 'alarm_identifier,alarm_name,ne_type,equip_info',
					'SERVERITY_IDS':'${alarm_severity}',
					'search_text':'${search_text}',
					'like_fields': 'alarm_serverity_value,alarm_identifier,alarm_name,ne_type,equip_info,event_type_value,deal_state,specific_problem',
					'ALARM_IDENTIFIER':'',
					'EVENT_TYPE':'',
					'PROBABLE_CAUSE':'',
					'NE_TYPE':'',
					'EQUIP_INFO':'',
					'DEAL_STATE':'',
					'EVENT_TIME_START':'',
					'EVENT_TIME_STOP':'',
					rd: ''
				},
				params_active:{      // 活动告警  -- 高级查询表单参数
					'SERVERITY_IDS':'${alarm_severity}',
					'ALARM_IDENTIFIER':'',
					'EVENT_TYPE':'',
					'PROBABLE_CAUSE':'',
					'NE_TYPE':'',
					'EQUIP_INFO':'',
					'DEAL_STATE':'',
					'EVENT_TIME_START':'',
					'EVENT_TIME_STOP':''
				},
				params_historyFault:{  //历史告警  -- 查询参数 
					timeZone : timeZone,
					like_fields : 'alarm_identifier,alarm_name,equip_info',
					'SERVERITY_IDS':'',
					'search_text':'',
					'HIS_ALARM_IDENTIFIER':'',
					'EVENT_TYPE':'',
					'PROBABLE_CAUSE':'',
					'NE_TYPE':'',
					'EQUIP_INFO':'',
					'DEAL_STATE':'',
					'EVENT_TIME_START':'',
					'EVENT_TIME_STOP':'',
					rd: ''
				},
				params_history:{     //历史告警  -- 高级查询表单参数 
					'SERVERITY_IDS':'',
					'ALARM_IDENTIFIER':'',
					'EVENT_TYPE':'',
					'PROBABLE_CAUSE':'',
					'NE_TYPE':'',
					'EQUIP_INFO':'',
					'DEAL_STATE':'',
					'EVENT_TIME_START':'',
					'EVENT_TIME_STOP':''
				},
				alarmLevel:[     // 告警级别下拉选择数据 
					{text:'<%=rb.getString("QuanBu")%>',id:''},
					{text:'<%=rb.getString("JinJiGaoJing")%>',id:'31001'},
					{text:'<%=rb.getString("ZhuYaoGaoJing")%>',id:'31002'},
					{text:'<%=rb.getString("CiYaoGaoJing")%>',id:'31003'},
					{text:'<%=rb.getString("JingGaoGaoJing")%>',id:'31004'}
				],
				eventType:[    //告警类型下拉选择数据 
					{text:'<%=rb.getString("QuanBu")%>',id:''},
					{text:'<%=rb.getString("TongXinGaoJing")%>',id:'30000'},
					{text:'<%=rb.getString("FuWuZhiLiangGaoJing")%>',id:'30001'},
					{text:'<%=rb.getString("ChuLiShiBaiGaoJing")%>',id:'30002'},
					{text:'<%=rb.getString("SheBeiGaoJing")%>',id:'30003'},
					{text:'<%=rb.getString("HuanJingGaoJing")%>',id:'30004'},
					{text:'<%=rb.getString("XingNengYiChuGaoJing")%>',id:'30006'}
				],
				deviceType:[],
				alarmStatusActive:[  //活动告警 -- 告警状态下拉选择数据 
					{text:'<%=rb.getString("QuanBu")%>',id:''},
					{text:'<%=rb.getString("WeiQueRenWeiQingChu")%>',id:'0'},
					{text:'<%=rb.getString("YiQueRenWeiQingChu")%>',id:'1'},
				],
				alarmStatusHistory:[  //历史告警 -- 告警状态下拉选择数据 
					{text:'<%=rb.getString("QuanBu")%>',id:''},
					{text:'<%=rb.getString("WeiQueRenYiQingChu")%>',id:'2'},
					{text:'<%=rb.getString("YiQueRenYiQingChu")%>',id:'3'},
				],
				rowData:[],
				dialogTitle:'',
				styleObj:{},
				confirmForm:{
					"confirmUser":'',
					"confirmTime":'',
					"description":''
				},
				showConfirmInfo:false,      //显示确认告警窗口参数。 
				confirmFlag:false,      //用处控制告警确认窗口中的确认人和确认时间信息的显隐  
				clearFlag:false,
				exportFlag:false,
				sortActive:'',
				orderActive:'',
				sortHistory:'',
				orderHistory:'',
				selectedIds:'',
				alarmSelected:'',
				selectedList:[], // 已选设备列表
				buttomOptShow:false, // 底部批量操作弹窗
				selectedAlarmShow:false,
				operationType:false,   // 判断是单个操作 还是批量操作
				showDeviceMsg:false
			}
			
		},
		computed:{
			// 底部批量操作弹窗
			bottomSliderCls() {
				var vm = this;
				return {
					'bottom-slider': true,
					show: vm.buttomOptShow
				};
			},
			// 查看设备弹窗
			selectedAlarmInfoCls() {
				return {
					'selected-alarm-info': true,
					show: this.selectedAlarmShow
				};
			},
			// 查看设备弹窗名称
			selectedTitle(){
				return '<%=rb.getString("PiLiangGaoJingYiXuanBiaoTi")%>';
			},
			// 权限控制 批量操作
			selectionShow() {
				var vm = this;
				
				return writableMap['CODE_ALARM_VIEW'] == true
			},
		},
		methods:{
			// 设备类型数据请求
			init(){
				var vm = this;
				axios.post("${ctx}/cell/fault/findFileterAlarmNETypeList.action").then(function(response){
    	    		vm.deviceType = response.data
    	    	})
    	    	
			},
			/**
			* 模糊查询 -- 活动告警 
			* @param val{string}   查询的val
			*/
			activeAlarmQuery(val){
				var vm =  this;
				vm.$refs.ctableActiveFault.reset()
				vm.params_activeFault.search_text = val;
				vm.params_activeFault.SERVERITY_IDS = vm.params_active.SERVERITY_IDS = "";
				vm.params_activeFault.ALARM_IDENTIFIER = vm.params_active.ALARM_IDENTIFIER = "";
				vm.params_activeFault.EVENT_TYPE = vm.params_active.EVENT_TYPE = "";
				vm.params_activeFault.PROBABLE_CAUSE = vm.params_active.PROBABLE_CAUSE = "";
				vm.params_activeFault.NE_TYPE = vm.params_active.NE_TYPE = "";
				vm.params_activeFault.EQUIP_INFO = vm.params_active.EQUIP_INFO = "";
				vm.params_activeFault.DEAL_STATE = vm.params_active.DEAL_STATE = "";
				vm.params_activeFault.EVENT_TIME_START = '';
    			vm.params_activeFault.EVENT_TIME_STOP = '';
    			vm.params_activeFault.rd = Math.random();
    			vm.params_active.time = [];
			},
			 // 高级查询  -- 活动告警 
			activeAlarmQueryAdvance(){  
				var vm = this,
					dateValue = vm.params_active.time;
				vm.$refs.ctableActiveFault.reset()
			
				vm.params_activeFault.search_text = "";
				vm.params_activeFault.SERVERITY_IDS = vm.params_active.SERVERITY_IDS;
				vm.params_activeFault.ALARM_IDENTIFIER = vm.params_active.ALARM_IDENTIFIER;
				vm.params_activeFault.EVENT_TYPE = vm.params_active.EVENT_TYPE;
				vm.params_activeFault.PROBABLE_CAUSE = vm.params_active.PROBABLE_CAUSE;
				vm.params_activeFault.NE_TYPE = vm.params_active.NE_TYPE;
				vm.params_activeFault.EQUIP_INFO = vm.params_active.EQUIP_INFO;
				vm.params_activeFault.DEAL_STATE = vm.params_active.DEAL_STATE;
    			vm.params_activeFault.rd = Math.random();
    			
				if(dateValue != null){
    				vm.params_activeFault.EVENT_TIME_START = dateValue[0];
	    			vm.params_activeFault.EVENT_TIME_STOP = dateValue[1];
    			}else{
    				vm.params_activeFault.EVENT_TIME_START = '';
	    			vm.params_activeFault.EVENT_TIME_STOP = '';
    			}
			},
			// 查询重置  -- 活动告警  
			activeAlarmReset(){
				var vm = this;
				
				vm.params_active.SERVERITY_IDS = "";
				vm.params_active.ALARM_IDENTIFIER = "";
				vm.params_active.EVENT_TYPE = "";
				vm.params_active.PROBABLE_CAUSE = "";
				vm.params_active.NE_TYPE = "";
				vm.params_active.EQUIP_INFO = "";
				vm.params_active.DEAL_STATE = "";
				vm.params_active.time = [];
			},
			/**
			* 模糊查询 -- 历史告警 
			* @param val{string}   查询的val
			*/
			historyAlarmQuery(val){
				var vm =  this;
				vm.$refs.ctableHistoryFault.reset() // 查询前跳转第一页
    			vm.params_historyFault.search_text  = val;
				vm.params_historyFault.SERVERITY_IDS = vm.params_history.SERVERITY_IDS = "";
				vm.params_historyFault.HIS_ALARM_IDENTIFIER = vm.params_history.ALARM_IDENTIFIER = "";
				vm.params_historyFault.EVENT_TYPE = vm.params_history.EVENT_TYPE = "";
				vm.params_historyFault.PROBABLE_CAUSE = vm.params_history.PROBABLE_CAUSE = "";
				vm.params_historyFault.NE_TYPE = vm.params_history.NE_TYPE = "";
				vm.params_historyFault.EQUIP_INFO = vm.params_history.EQUIP_INFO = "";
				vm.params_historyFault.DEAL_STATE = vm.params_history.DEAL_STATE = "";
				vm.params_historyFault.EVENT_TIME_START = "";
    			vm.params_historyFault.EVENT_TIME_STOP = "";
    			vm.params_historyFault.rd = Math.random();
    			vm.params_history.time = [];
			},
			// 高级查询  -- 历史告警
			historyAlarmQueryAdvance(){    
				var vm = this,
					dateValueHis = vm.params_history.time;
				vm.$refs.ctableHistoryFault.reset()
				vm.params_historyFault.search_text = "";
				vm.params_historyFault.SERVERITY_IDS = vm.params_history.SERVERITY_IDS;
				vm.params_historyFault.HIS_ALARM_IDENTIFIER = vm.params_history.ALARM_IDENTIFIER;
				vm.params_historyFault.EVENT_TYPE = vm.params_history.EVENT_TYPE;
				vm.params_historyFault.PROBABLE_CAUSE = vm.params_history.PROBABLE_CAUSE;
				vm.params_historyFault.NE_TYPE = vm.params_history.NE_TYPE;
				vm.params_historyFault.EQUIP_INFO = vm.params_history.EQUIP_INFO;
				vm.params_historyFault.DEAL_STATE = vm.params_history.DEAL_STATE;
    			vm.params_historyFault.rd = Math.random();
				
    			if(dateValueHis != null){
    				vm.params_historyFault.EVENT_TIME_START = dateValueHis[0];
	    			vm.params_historyFault.EVENT_TIME_STOP = dateValueHis[1];
    			}else{
    				vm.params_historyFault.EVENT_TIME_START = '';
	    			vm.params_historyFault.EVENT_TIME_STOP = '';
    			}
			},
			// 查询重置  -- 历史告警 
			historyAlarmReset(){   
				var vm = this;
				
				vm.params_history.SERVERITY_IDS = "";
				vm.params_history.ALARM_IDENTIFIER = "";
				vm.params_history.EVENT_TYPE = "";
				vm.params_history.PROBABLE_CAUSE = "";
				vm.params_history.NE_TYPE = "";
				vm.params_history.EQUIP_INFO = "";
				vm.params_history.DEAL_STATE = "";
				vm.params_history.time = [];
			},
			handleClick(){
				var vm = this;
				vm.delAllSelectedRecord();
				vm.$nextTick(()=>{
					document.body.click();
					vm.$refs.ctableActiveFault.refresh();//表格刷新
					vm.$refs.ctableHistoryFault.refresh();//表格刷新
				});
				
			},
			//点击页面其他地方菜单收起
			handerClose(){ 
		        this.$refs.menuActiveFault.hide();
		        this.$refs.menuHistoryFault.hide();
		    },
			/**
			* 告警列表选中
			* @param selection{Array}   选中数据
			*/
			alarmSelect(selection){
				var vm = this;
				var tb = vm.$refs.ctableActiveFault;
				vm.alarmSelected = selection;
				if(vm.alarmSelected.length>5){

				}
				if(vm.alarmSelected && vm.alarmSelected.length>0){
					vm.selectedList = selection.map(function(value){
						var rowItem = {alarm_id: value};
						return rowItem;
					})
					vm.buttomOptShow = true;
					
				}else{
					vm.selectedList = [];
					vm.buttomOptShow = false;
					vm.selectedAlarmShow = false;
				}
			},
			// 清除当前tab的单个设备选择记录
			delSingleRecord(row) { 
				var vm = this,
					rowkey = 'alarm_id';
					
				if(vm.faultType == 'activeFault'){
					var tb = vm.$refs.ctableActiveFault;
		    	}else if(vm.faultType == 'historyFault'){
		    		var tb = vm.$refs.ctableHistoryFault;
		    	}
				var rows = tb.getData();
				var cklist = tb.getChecked();
				rows.map(function(item,index){
					if(item[rowkey] == row[rowkey].alarm_id) tb.toggleRowSelection(item, false);
				});
				
				cklist.splice(cklist.indexOf(row[rowkey]),1);
				
				vm.selectedList = vm.selectedList.filter(function(item){
					return item[rowkey] != row[rowkey];
				});
			},
			// 清除当前tab的所有设备选择记录
			delAllSelectedRecord(){ 
				var vm = this;
				vm.$refs.ctableActiveFault.clearSelection();
				vm.$refs.ctableHistoryFault.clearSelection();
			},
			//批量删除
			deleteBatch(){
				var vm = this;
				var	alarmIds=[];
				vm.alarmSelected.map((item,index) => {
    	    		alarmIds.push({alarm_id:item.alarm_id,small_cell_code:item.small_cell_code})
    	    		return alarmIds;
    	    	})
				let alarmIdsData = JSON.stringify(alarmIds);
		    	vm.$confirm('<%=rb.getString("QueRenPiLiangShanChuGaoJing")%>','<%=rb.getString("QueRen")%>',{
		    		customClass:'warningConfirm',
		    		confirmButtonText:'<%=rb.getString("QueDing")%>',
		    		cancelButtonText:'<%=rb.getString("QuXiao")%>',
		    	}).then(function(){
		    		axios.post('${ctx}/cell/fault/batchClearHistoryAlarm.action',alarmIdsData,{headers:{'Content-Type':'application/json;charset=utf-8'},}).then(function(response){
			    		var data = response.data;
			    		if(data["success"]){
							vm.delAllSelectedRecord();
			    			vm.$message.success('<%=rb.getString("ChengGong")%>')
			    			vm.$refs.ctableHistoryFault.refresh();//表格刷新
			    		}else{
			    			vm.$message.error('<%=rb.getString("ShanChuGaoJingShiBai")%>') //错误提示信息 
			    		}
			    	}) 
		    	}).catch(function(){
		    		
		    	})
			},
			//批量清除 打开批量清除弹窗
			clearBatch(){
				var vm = this;
		    	vm.clearFlag = true;
				vm.operationType = true;
		    	vm.showConfirmInfo = true;
				vm.styleObj = {
		    				'marginTop':'20vh'
		    		}
		    	vm.dialogTitle = '<%=rb.getString("QueRen")%>';
		    	
			},
			//批量确认 打开批量确认弹窗
			confirmBatch(){
				var vm = this;
		    	vm.clearFlag = false;
				vm.operationType = true;
				vm.confirmFlag = false;
				vm.styleObj = {
		    				'marginTop':'20vh'
		    		}
		    	vm.showConfirmInfo = true;
		    	vm.dialogTitle = '<%=rb.getString("QueRenGaoJing")%>';
			},
			// 批量反确认
			unConfirmBatch(){
				var vm = this,
					type,
	    			faultType = vm.faultType,
	    			tableRef = {
		    			'active':'ctableActiveFault',
		    			'his':'ctableHistoryFault'
		    	};
				
				if(faultType == 'activeFault'){	    		
					type = 'active';
		    	}else if(faultType == 'historyFault'){
		    		type = 'his'
		    	}
				var	alarmIds=[];
				vm.alarmSelected.map((item,index) => {
    	    		alarmIds.push({alarm_id:item.alarm_id,type:type})
    	    		return alarmIds;
    	    	})
				let alarmIdsData = JSON.stringify(alarmIds);
		    	vm.$confirm('<%=rb.getString("QueDingQuXiaoGaoJingQueRen")%>','<%=rb.getString("QueRen")%>',{
		    		customClass:'warningConfirm',
		    		confirmButtonText:'<%=rb.getString("QueDing")%>',
		    		cancelButtonText:'<%=rb.getString("QuXiao")%>',
		    	}).then(function(){
		    		axios.post('${ctx}/cell/fault/batchCancelConfirmAlarm.action',alarmIdsData,{headers:{'Content-Type':'application/json;charset=utf-8'},}).then(function(response){
			    		var data = response.data;
			    		if(data["success"]){
							vm.delAllSelectedRecord();
			    			vm.$message.success('<%=rb.getString("ChengGong")%>');
			    			vm.$refs[tableRef[type]].refresh();//表格刷新
			    		}else{
			    			vm.$message.error('<%=rb.getString("FanQueRenGaoJingTiShi")%><%=rb.getString("TiShiShiBai")%>') //错误提示信息
			    		}
			    	})
		    	}).catch(function(){
		    		
		    	})
			},
			// 取消批量操作
			cancelBatchOpt(){ 
				var vm = this;
				vm.buttomOptShow = false;
				vm.selectedAlarmShow = false;
				vm.delAllSelectedRecord();
			},
			
			/**
			* 表格数据点击出现menus菜单事件
			* @param row{object}   行数据
			* @param ev{object}   event数据
			*/
			optClick(row,ev){ // 操作项： 1.详细  2.确认告警  3.反确认告警  4.清除告警  5.删除告警  
		    	var vm = this,
		    		faultType = vm.faultType ,
		    		status = row.deal_state,
		    		confirmFlag = true ,
		    		clearFlag = false;   //清除状态 
		    		
				if(status == '0' || status == '2'){
					confirmFlag = false
				}
				
				if(status == '2' || status == '3'){
					clearFlag = true;
				}
			
		    	vm.rowData = row;
		    	vm.menus= [
					{label:'<%=rb.getString("XiangXi")%>',cls:"el-icon-operation-info el-icon",code:'detail'},
					{label:'<%=rb.getString("QueRenGaoJing")%>',cls:"el-icon-operation-confirm el-icon CODE_ALARM_VIEW hidden" ,code:'confirm'},
					{label:'<%=rb.getString("FanQueRenGaoJing")%>',cls:"el-icon-operation-unconfirm el-icon CODE_ALARM_VIEW hidden" ,code:'unConfirm',disable:!confirmFlag},
					{label:'<%=rb.getString("QingChuGaoJing")%>',cls:"el-icon-operation-clear el-icon CODE_ALARM_VIEW hidden",code:'clear',show:!clearFlag},
					{label:'<%=rb.getString("ShanChuGaoJing")%>',cls:"el-icon-operation-delete el-icon CODE_ALARM_VIEW hidden",code:'del',show:clearFlag}
				]
		    	
		    	vm.$nextTick(function(){
		    		document.body.click();

					if(faultType == 'activeFault'){	    		
						vm.$refs.menuActiveFault.show(ev);
			    	}else if(faultType == 'historyFault'){
			    		vm.$refs.menuHistoryFault.show(ev);
			    	}
		    	});
		    },
			/**
			* 菜单点击事件
			* @param ev{object}   菜单点击列表的信息
			*/
		    clickMenu(ev){
		    	var codes = {
		    		detail:this.detailAlarmInfo,
		    		confirm:this.openConfirmDialog,
		    		unConfirm:this.unconfirmAlarm,
		    		clear:this.clearAlarm,
		    		del:this.deleteAlarm
		    	}
		    	if(codes[ev.code]){
		    		codes[ev.code](this.$root.rowData)
		    	}
		    },
			/**
			* 打开告警详情页面
			* @param row{object}   行数据
			*/  
		    detailAlarmInfo(row){
				var vm = this,
					type,
	    			faultType = vm.faultType;
				
				if(faultType == 'activeFault'){	    		
					type = 'active';
		    	}else if(faultType == 'historyFault'){
		    		type = 'his'
		    	}
				vm.detailUrl = "${ctx}/cell/fault/goAlarmDetail.action";
				
				vm.$refs.slide.showSlide(function(){
	    	    	vm.modal = false;
	    	    	eventBus.$emit('detail-info',type,row.alarm_id);
	    	    });
		
		    },
			// 关闭告警详情页面
		    closeDetailFault(){     
		    	this.$refs.slide.hide();
		    },
			/**
			* 打开告警确认窗口 
			* @param row{object}   行数据
			*/  
		    openConfirmDialog(row){
		    	var vm = this,type= '',
		    		faultType = vm.faultType
					status = row.deal_state;
		    	
		    	if(faultType == 'activeFault'){	    		
					type = 'active';
		    	}else if(faultType == 'historyFault'){
		    		type = 'his'
		    	}
		    
		    	vm.clearFlag = false;
		    	vm.operationType = false;
		    	if(status == '1' || status == '3'){
		    		vm.confirmFlag = true;
		    		vm.styleObj = {
		    				'marginTop':'15vh'
		    		}
		    	}else{
		    		vm.confirmFlag = false;
		    		vm.styleObj = {
		    				'marginTop':'20vh'
		    		}
		    	}
		    	vm.dialogTitle = '<%=rb.getString("QueRenGaoJing")%>';
		    	vm.showConfirmInfo = true;
		    	
		    	axios.post('${ctx}/cell/fault/queryAlarmDetail.action',stringify({
	    			alarm_id : vm.rowData.alarm_id,
	    			type:type,
	    			timeZone:vm.timeZone
		    	})).then(function(response){
		    		var data = response.data;
					vm.confirmForm.confirmUser = data.DEAL_USER;
					vm.confirmForm.confirmTime = data.DEAL_TIME;
					vm.confirmForm.description = data.DEAL_MEMO;
		    	}) 
		    	
		    },
			//告警弹窗 确认方法 
		    confirmAlarm(){   
		    	var vm = this,
					type,msg,
	    			faultType = vm.faultType,
	    			tableRef = {
		    			'active':'ctableActiveFault',
		    			'his':'ctableHistoryFault'
		    	};
				
				if(faultType == 'activeFault'){	    		
					type = 'active';
		    	}else if(faultType == 'historyFault'){
		    		type = 'his'
		    	}
				if(!vm.operationType){
					if( vm.clearFlag == true){
						url = '${ctx}/cell/fault/clearAlarm.action';    //清除告警 
						msg = '<%=rb.getString("QingChuGaoJingTiShi")%>'
					}else {
						url = '${ctx}/cell/fault/confirmAlarm.action';     //确认告警 
						msg = '<%=rb.getString("QueRenGaoJingTiShi")%>'
					}
					
					axios.post(url,stringify({
						alarm_id : vm.rowData.alarm_id,
						small_cell_code: vm.rowData.small_cell_code,
						type : type,
						text : vm.confirmForm.description
					})).then(function(response){
						var data = response.data;
						if(data["success"]){
							vm.closeConfirmInfo();
							vm.$message.success( '<%=rb.getString("ChengGong")%>');
							vm.$refs[tableRef[type]].refresh();//表格刷新
						}else{
							vm.closeConfirmInfo();
							vm.$message.error(msg + '<%=rb.getString("TiShiShiBai")%>') //错误提示信息
						}
					}) 
				}else{
					var	alarmIds=[];
					vm.alarmSelected.map((item,index) => {
						alarmIds.push({alarm_id:item.alarm_id,small_cell_code:item.small_cell_code,type:type,text:vm.confirmForm.description})
						return alarmIds;
					})
					if( vm.clearFlag == true){
						
						url = '${ctx}/cell/fault/batchClearAlarm.action';    //批量 清除告警 
						msg = '<%=rb.getString("QingChuGaoJingTiShi")%>'
					}else {
						url = '${ctx}/cell/fault/batchConfirmAlarm.action';     // 批量 确认告警 
						msg = '<%=rb.getString("QueRenGaoJingTiShi")%>'
					}
					let alarmIdsData = JSON.stringify(alarmIds);
					axios.post(url,alarmIdsData,{headers:{'Content-Type':'application/json;charset=utf-8'},}).then(function(response){
						var data = response.data;
						if(data["success"]){
							vm.closeConfirmInfo();
							vm.delAllSelectedRecord();
							vm.$message.success( '<%=rb.getString("ChengGong")%>');
							vm.$refs[tableRef[type]].refresh();//表格刷新
						}else{
							vm.closeConfirmInfo();
							vm.$message.error(msg + '<%=rb.getString("TiShiShiBai")%>') //错误提示信息
						}
					}) 
				 }
				
				
		    },
			// 关闭告警确认窗口
		    closeConfirmInfo(){

		    	this.showConfirmInfo = false;
				this.confirmForm = {
					"confirmUser":'',
					"confirmTime":'',
					"description":''
				}
		    },
			// 浮窗关闭事件
			dialogClose(){
				this.confirmForm = {
					"confirmUser":'',
					"confirmTime":'',
					"description":''
				}
			},
			/**
			* 告警反确认
			* @param row{object}   行数据
			*/ 
		    unconfirmAlarm(row){  
		    	var vm = this,
					type,
	    			faultType = vm.faultType,
	    			tableRef = {
		    			'active':'ctableActiveFault',
		    			'his':'ctableHistoryFault'
		    	};
				vm.operationType = false;
				if(faultType == 'activeFault'){	    		
					type = 'active';
		    	}else if(faultType == 'historyFault'){
		    		type = 'his'
		    	}
		
		    	vm.$confirm('<%=rb.getString("QueDingQuXiaoGaoJingQueRen")%>','<%=rb.getString("QueRen")%>',{
		    		customClass:'warningConfirm',
		    		confirmButtonText:'<%=rb.getString("QueDing")%>',
		    		cancelButtonText:'<%=rb.getString("QuXiao")%>',
		    	}).then(function(){
		    		axios.post('${ctx}/cell/fault/cancelConfirmAlarm.action',stringify({
		    			alarm_id : row.alarm_id,
		    			type:type
			    	})).then(function(response){
			    		var data = response.data;
			    		if(data["success"]){
			    			vm.$message.success('<%=rb.getString("ChengGong")%>');
			    			vm.$refs[tableRef[type]].refresh();//表格刷新
			    		}else{
			    			vm.$message.error('<%=rb.getString("FanQueRenGaoJingTiShi")%><%=rb.getString("TiShiShiBai")%>') //错误提示信息
			    		}
			    	})
		    	}).catch(function(){
		    		
		    	})
		    },
			/**
			* 打开告警清除窗口
			* @param row{object}   行数据
			*/ 
		    clearAlarm(row){
		    	var vm = this,type= '',
					status = row.deal_state,
					faultType = vm.faultType;
	    		
		    	vm.clearFlag = true;
	    		vm.operationType = false;	
		    	if(status == '1' || status == '3'){
		    		vm.confirmFlag = true;
					vm.styleObj = {
		    			'marginTop':'15vh'
		    		}
		    	}else{
					vm.styleObj = {
		    			'marginTop':'15vh'
		    		}
		    		vm.confirmFlag = false;    
		    	}
		    	vm.showConfirmInfo = true;
		    	vm.dialogTitle = '<%=rb.getString("QueRen")%>';
		    	
		    	if(faultType == 'activeFault'){	    		
					type = 'active';
		    	}else if(faultType == 'historyFault'){
		    		type = 'his'
		    	}
		    	
		    	axios.post('${ctx}/cell/fault/queryAlarmDetail.action',stringify({
	    			alarm_id : vm.rowData.alarm_id,
	    			type:type,
	    			timeZone:vm.timeZone
		    	})).then(function(response){
		    		var data = response.data;
					vm.confirmForm.description = data.DEAL_MEMO;
		    	}) 
		    },
			/**
			* 删除告警
			* @param row{object}   行数据
			*/ 
		    deleteAlarm(row){  
		    	var vm = this,
					alarmId = row.alarm_id;
				vm.operationType = false;
		    	vm.$confirm('<%=rb.getString("QueRenShanChuGaoJing")%>','<%=rb.getString("QueRen")%>',{
		    		customClass:'warningConfirm',
		    		confirmButtonText:'<%=rb.getString("QueDing")%>',
		    		cancelButtonText:'<%=rb.getString("QuXiao")%>',
		    	}).then(function(){
		    		axios.post('${ctx}/cell/fault/clearHistoryAlarm.action',stringify({
		    			alarm_id : alarmId,
						small_cell_code:vm.rowData.small_cell_code,
			    	})).then(function(response){
			    		var data = response.data;
			    		if(data["success"]){
			    			vm.$message.success('<%=rb.getString("ChengGong")%>')
			    			vm.$refs.ctableHistoryFault.refresh();//表格刷新
			    		}else{
			    			vm.$message.error('<%=rb.getString("ShanChuGaoJingShiBai")%>') //错误提示信息 
			    		}
			    	}) 
		    	}).catch(function(){
		    		
		    	})
		    	
		    },
			// 打开导出数据选择设备组页面
			openExportDialg(){     
				this.exportFlag = true;
			},
			/**
			* 导出 设备组列表选中数据
			* @param val{Array}   导出设备组选中列表
			*/ 
			deviceGroupSelect(val){
				var vm = this;
				setTimeout(function(){
					var selection = vm.$refs.ctableDeviceGroup.getChecked();
					vm.selectedIds = selection;
		    	},10)
		    	vm.showDeviceMsg = false;
			},
			// 导出确认
			exportAlarm(){ 
				var vm = this,
					faultType = vm.faultType,
	    			params={timeZone: timeZone},
	    			urls = {
		    			'activeFault':'${ctx}/cell/fault/exportActiveAlarmsToCSV.action',
		    			'historyFault':'${ctx}/cell/fault/exportHistoryAlarmsCSV.action'
		    		};
				
				var selection = vm.$refs.ctableDeviceGroup.getChecked();
				if(selection.length == 0){
					vm.showDeviceMsg = true;
				}else{
					vm.showDeviceMsg = false;
				}
				if(!vm.showDeviceMsg){
					if (faultType == 'activeFault'){
						params.active_alarms_search_text = vm.params_activeFault.search_text;
						params.active_alarms_like_fields = vm.params_activeFault.like_fields;
						params.active_alarms_identifier = vm.params_activeFault.ALARM_IDENTIFIER;
						params.active_alarms_event_type = vm.params_activeFault.EVENT_TYPE;
						params.active_alarms_severity = vm.params_activeFault.SERVERITY_IDS;
						params.active_alarms_event_time_start = vm.params_activeFault.EVENT_TIME_START;
						params.active_alarms_event_time_stop = vm.params_activeFault.EVENT_TIME_STOP;
						params.active_alarms_probable_cause = vm.params_activeFault.PROBABLE_CAUSE;
						params.active_alarms_ne_type = vm.params_activeFault.NE_TYPE;
						params.active_alarms_deal_state = vm.params_activeFault.DEAL_STATE;
						params.active_alarms_equip_info = vm.params_activeFault.EQUIP_INFO;
						
						params.sort = vm.sortActive;
						params.order = vm.orderActive;
					}else{

						params.his_alarms_search_text = vm.params_historyFault.search_text;
						params.his_alarms_like_fields = vm.params_historyFault.like_fields;
						params.his_alarms_identifier = vm.params_historyFault.HIS_ALARM_IDENTIFIER;
						params.his_alarms_event_type = vm.params_historyFault.EVENT_TYPE;
						params.his_alarms_severity = vm.params_historyFault.SERVERITY_IDS;
						                                    
						params.his_alarms_event_time_start = vm.params_historyFault.EVENT_TIME_START;
						params.his_alarms_event_time_stop = vm.params_historyFault.EVENT_TIME_STOP;
						params.his_alarms_probable_cause = vm.params_historyFault.PROBABLE_CAUSE;
						params.his_alarms_ne_type = vm.params_historyFault.NE_TYPE;
						params.his_alarms_deal_state = vm.params_historyFault.DEAL_STATE;
						params.his_alarms_equip_info = vm.params_historyFault.EQUIP_INFO;
						
						params.sort = vm.sortHistory;
						params.order = vm.orderHistory;
					}
					params.group_ids = (vm.selectedIds).join(",");
					exportByForm(urls[faultType],params);
				}
			},
			// 取消导出
			closeExport(){ 
				this.exportFlag = false;
				this.$refs.ctableDeviceGroup.clearSelection();
				this.showDeviceMsg = false;
			},
			/**
			* 记录排序信息  -- 活动告警 
			* @param data{object}   排序信息
			*/ 
			sortChangeActive(data){
				var vm =this,
					order={
						ascending:'asc',
						descending:'desc'
				};
				vm.sortActive = data.prop;
				vm.orderActive = order[data.order];
			},
			/**
			* 记录排序信息  -- 历史告警 
			* @param data{object}   排序信息
			*/ 
			sortChangeHistory(data){
				var vm =this,
					order={
						ascending:'asc',
						descending:'desc'
				};
				vm.sortHistory = data.prop;
				vm.orderHistory = order[data.order];
			}
		},
		mounted(){
			this.init();
		}
	});
</script>