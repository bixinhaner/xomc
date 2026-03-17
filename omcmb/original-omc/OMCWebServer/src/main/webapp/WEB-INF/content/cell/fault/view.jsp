<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style type="text/css">
	#alarmViewPage{
		height:100%;
		overflow:hidden;
		min-width: 900px;
		border-radius: 4px;
	}
	#alarmViewPage .alarmMain{
		display: flex;
		position: absolute;
		top: 0px;
		bottom: 0px;
		left: 0px;
		right: 0px;
		background: #FFFFFF;
	}
	#alarmViewPage .alarmMain .concentLeft{
		width: 240px;
		display: flex;
		flex-direction: column;
		background: #FFFFFF;
		border-right: 1px solid #E9E9E9;
	}
	#alarmViewPage .concentLeft .totalTitle{
		height: 50px;
		line-height: 50px;
		font-size: 14px;
		padding-left: 20px;
		font-weight: 550;
		border-bottom: 1px solid #E9E9E9;
		border-top: 1px solid #E9E9E9;
	}
	#alarmViewPage .concentLeft .alarmTotal .totalTitle{
		border-bottom:none;
	}
	#alarmViewPage .concentLeft .totalTitle span:nth-child(1){
		font-size: 12px;
		margin: 0px 6px 0px 10px;
	}
	#alarmViewPage .alarmMain .concentRight{
		background: #FFFFFF;
		flex-grow:1;
		position: relative;
		overflow: auto;
	}
	#alarmViewPage .concentRight .el-query{
		flex-grow:1;
	}
	#alarmViewPage .queryInfo{
		display:inline-block;
		margin-right:35px;
	}
	#alarmViewPage .queryInfo label{
		display:block;
		line-height:24px;
	}
	#alarmViewPage .alarmTemplate {
		height: calc(100% - 98px)!important;
		width: 240px;
		display: flex;
		flex-direction: column;
		border-right:1px solid #E9E9E9; 
	}
	#alarmViewPage .addTemplate{
		position: absolute;
		right: 10px;
		top: 10px;
	}
	
	.alarmMinor .el-icon:before{
		color: #FFDA41;
		font-size: 20px;
	}
	.alarmMajor .el-icon:before{
		color: #FF973E;
		font-size: 20px;
	}
	.alarmCritical .el-icon:before{
		color: #FC5959;
		font-size: 20px;
	}
	.alarmWarning .el-icon:before{
		color: #60BEFC;
		font-size: 20px;
	}
	#alarmViewPage .templateTableOne{
		height: calc(100% - 50px)!important;
		overflow: auto;
		width: 240px;
	}
	#alarmViewPage .templateTableOne .el-table__body-wrapper{
		height: 100%!important;
	}
	#alarmViewPage .templateNameSty .el-icon:before{
		color: #F2B354;
		font-size: 14px;
	}
	#alarmViewPage .templateNameSty div:first-child{
		max-width: 160px;
		margin-right:10px;
		overflow: hidden;
		white-space: nowrap;
		text-overflow: ellipsis;
	}
	#alarmViewPage .toolBarLeftName{
		height:36px;
		line-height:35px;
		padding-left:10px;
		font-size: 14px;
		font-weight: 550;
		overflow: hidden;
		white-space: nowrap;
		text-overflow: ellipsis;
		border-bottom: 1px solid #DFE2EE;
	}
	#alarmViewPage .packUpSty{
		position: absolute;
		height: 30px;
		width: 15px;
		display: flex;
		align-items: center;
		border-radius:10px 10px;
		left: 0px;
		top: 400px;
		z-index: 20;
	}
	#alarmViewPage .packUpSty .el-icon:before{
		color: #FFFFFF;
		font-size: 12px;
	}
	#alarmViewPage .templateTableOne .el-table--border{
		border:none;
	}
	#echartsDetail .el-dialog__body{
		padding: 0!important;
	}
	#echartsDetail .el-icon-close{
		font-size: 16px!important;
		position: absolute!important;
		top: 5px!important;
		right: 0px!important;
	}
	#echartsDetail .detailExport{
		font-size: 16px;
		position: absolute;
		top: 13px;
		right: 50px;
	}
	#alarmViewPage .templateTableOne .el-table__row td:nth-child(2) .cell{
		margin-left: -4px;
	}
	#alarmViewPage .templateTableOne .cmenu{
		z-index: 361!important;
	}
	#alarmViewPage .slide-position-top .slide-content{
		padding: 0px;
	}
	.unconfirmInactive .el-icon:before{
		color: #E88282;
		font-size: 20px;
	}
	.confirmInactive .el-icon:before{
		color: #E88282;
		font-size: 20px;
	}
	.unconfirmActive .el-icon:before{
		color: #67D972;
		font-size: 20px;
	}
	.confirmActive .el-icon:before{
		color: #67D972;
		font-size: 20px;
	}
	.el-time-panel{
		left: -25px!important;
	}
	.el-tooltip__popper{
		max-width: 800px;
	}
	.severityPopperCls .severityCls >div{
		height: 28px;
		width: 140px;
		display: flex;
		padding-left: 10px;
		align-items: center;
		cursor: pointer;
	}
	.severityPopperCls{
		padding: 0px!important;
	}
	.severityPopperCls .severityCls >div:hover{
		background-color: #E9E9E9;
	}
	.no-outline:focus {
		outline: none ;
	}
	.operationNoBubCls{
		z-index: 999
	}
	.exportTimeBoxCls{
		display: flex;
		padding-top: 20px;
		margin-left: 20px;
		align-items: center;
	}
	.queryTimeCls .el-button--text{
		display: none;
	}
	#alarmViewPage  .grayIcon::before{
		color: #7A7992;
		font-size: 18px;
	}
	#alarmViewPage .specialQueryBoxCls .headQueryBox .queryGroup input {
		width: 260px; 
	}
    #alarmViewPage .specialQueryBoxCls .headQueryInputTypeBoxCls  .el-select>.el-input{
        width: 140px;
    }
    #alarmViewPage .specialQueryBoxCls .headQueryInputTypeBoxCls  .el-select .el-input__inner{
        width: 140px;
    }
    #alarmViewPage .el-date-editor .el-range__close-icon ,#alarmViewPage .el-date-editor .el-icon-circle-close{
        line-height:20px;
        font-size: 12px;
    }
    #alarmViewPage .isFilterSiteAlarmCls{
        position: absolute;
        right: 10px;
        top: 12px;
        display: flex;
        align-items: center;
    }
</style>
<div class="overflow-cls">
<div id="alarmViewPage" class="panelDefault">
	<div class="container">
		<el-tabs class="fit newTabs" v-model="activeName" style='height:100%' @tab-click="tabClick">
			<el-tab-pane v-if="alarmViewTabShow" label="<%=rb.getString("AlarmView")%>" name="alarmView">
				<div :class="loading ? 'alarmMain loading' : 'alarmMain'">
					<div class="concentLeft" v-show="!packUpClickType">
							<div class="alarmTotal">
									<div class="totalTitle"><%=rb.getString("ZongLiang")%></div>
									<el-radio-group size="mini" v-model='totalChildNum ' class="commonRadioButton" @change="totalClick" style='margin: 0 10px 10px;'>
										<el-radio-button label="1"><%=rb.getString("QuanBu")%></el-radio-button>
										<el-radio-button label="2"><%=rb.getString("HuoDong")%></el-radio-button>
										<el-radio-button  label="3"><%=rb.getString("LiShi")%></el-radio-button>
									</el-radio-group>
							</div>
							<div class="alarmTemplate">
								<div class="totalTitle" style="position: relative;">
										<%=rb.getString("MuBan")%>
										<div class="newIconBoxCls-bt" v-if='alarmViewOptShow' style="right:10px;top:10px;" @click="addTemplate" tip="">
											<i class="el-icon el-icon-plus" ></i>
										</div>
								</div>
								<div class="templateTableOne">
									<el-ctable id="ctableTemplate" ref="ctableTemplate" :height="height" :show-pager="false" :show-header=false highlight-current-row="true"  :url="ctableTemplateUrl" :row-key="'templateId'"
										:time="6" :page-size="templatePageSize" :page-list="templatePageList" :pagination="true" :rownumber=false @row-click="selectTemplate">
										
										<el-table-column label='' width="200" prop="template_name">
											<template slot-scope="scope">
												<div style="display:flex;align-items:center;" class="templateNameSty">
													<div>{{scope.row.templateName}}</div>
													<div v-if="scope.row.emailEnable == '1' " class="el-icon el-icon-status-email" style="cursor:default;"></div>
												</div>
											</template>
										</el-table-column>
										<el-table-column label='' width="35" >
												<template slot-scope="scope">
													<div class="el-icon el-icon-operation-more-circle operationNoBubCls" @click="templateOptClick(scope.row,event)" v-clickoutside="handerClose" style="cursor: pointer;"></div>
												</template>
										</el-table-column>
							
									</el-ctable>
									<el-cmenu ref="menusTemplate" :data="menusTemplate" @click="clickTemplateMenu"></el-cmenu>
								</div>
							</div>
					</div>
					<div class="concentRight" >
						<!-- 按钮  -- 收起、展开 -->
						<div class="packUpSty mainColBg">
							<span v-if="!packUpClickType" class="el-icon el-icon-left" @click="packUpClick"></span>
							<span v-if="packUpClickType" class="el-icon el-icon-right" @click="packUpClick"></span>
						</div>
						<!-- 按钮  -- 通知设置 -->
						<div class="newIconBoxCls-bt" v-if='alarmViewOptShow' style="right:128px;top:12px;" @click="goAlarmNoticePage" tip="Notice">
							<span class="el-icon el-icon-circle-alarm"></span>
						</div>
						<!-- 按钮  -- 过滤 -->
						<div class="newIconBoxCls-bt" style="right:92px;top:12px;" @click="goFilterPage" tip="<%=rb.getString("GuoLv")%>">
							<span class="el-icon el-icon-operation-alarm-filter" ></span>
						</div>
						
						<!-- 按钮  -- 统计图表 -->
						<div class="newIconBoxCls-bt" style="right:56px;top:12px;" @click="goAlarmCharts" tip="<%=rb.getString("TongJi")%>">
							<span class="el-icon el-icon-circle-chart" ></span>
						</div>
							
						<!-- 按钮  -- 导出 -->
						<div class="newIconBoxCls-bt" style="right:20px;top:12px;" v-show="totalChildNum == '4'" tip="<%=rb.getString("DaoChu")%>">		
							<span class="el-icon-operation-export el-icon" @click="openExportDialg"></span>
						</div>
							
						<!-- 所有 活动 历史 按钮  -- 导出 -->
						<el-popover placement="bottom">
							<div style="width:600px;height:540px;position:relative">
								<div class="cardHeader">
									<span><%=rb.getString("SheBeiZu")%></span> <span class="slideIcon el-icon el-icon-close" @click="closeExport"></span>
								</div>
								<el-ctable ref="ctableDeviceGroup" height="350px" url="${ctx}/cell/fault/getDeviceGroup.action" @selection-change="deviceGroupSelect"
									:page-size="pageSize" pagination="true" :rownumber=true row-key="groupId" style="border-bottom:1px solid #E9E9E9;">
									<!-- 主列表 -->
									<el-table-column type="selection"></el-table-column>
									<el-table-column label='<%=rb.getString("SheBeiZuMingCheng")%>'  prop="groupName" ></el-table-column>
								</el-ctable>
								<p style='color:#FA5555;margin:0px 0px 10px 5px;' v-show="showDeviceGroupNotSelectMsg"><%=rb.getString("QingXuanZeSheBeiZu")%></p>
								<div class="exportTimeBoxCls">
									<label style="margin-right:10px;"><%=rb.getString("GuZhangShiJian")%>:</label> <!-- :picker-options="pickerOptions"-->
									<el-date-picker 
										size="mini" 
										v-model="exportTime" 
										value-format="yyyy-MM-dd HH:mm:ss" 
										type="datetimerange" 
										range-separator="—"  
										start-placeholder='<%=rb.getString("KaiShiShiJian")%>' 
										end-placeholder='<%=rb.getString("JieShuShiJian")%>'
                                        :picker-options="pickerOptions"
                                        @change="exportTimeChange"
									></el-date-picker>
								</div>
                                <p style='color:#FA5555;margin:0px 0px 10px 105px;' v-show="showExportTimeNotSelectMsg"><%=rb.getString("QingXuanZeShiJian")%></p>
								<span slot="footer" style="position:absolute;left:10px;bottom:10px;">
									<el-button type="primary" @click="exportAlarm"><%=rb.getString("QueDing")%></el-button>
									<el-button @click="closeExport"><%=rb.getString("QuXiao")%></el-button>
								</span>	
							</div>
							<div slot="reference" class="newIconBoxCls-bt" style="right:20px;top:12px;" v-show="totalChildNum !== '4'" @click="openExportDialg" tip="<%=rb.getString("DaoChu")%>">		
								<span class="el-icon-operation-export el-icon" ></span>
							</div>
						</el-popover>
						<el-ctable id="viewAlarmTable" 
							ref="ctableAlarm" :height="height"  
							:url="alarmDetailUrl"  
							@selection-change='alarmSelect'
							:query-params="params_alarm" 
							@load-success="tableLoadSuccess" 
							:page-size="pageSize" 
							:page-list="pageList" 
							:limit="limitAlarm" 
							pagination="true" 
							:rownumber=true  
							:time="6"
							@sort-change="sortChangeActive" 
							@row-click="alarmRowClick"
							:row-key="'alarm_id'">
							<template slot="toolbar" >
								<div style="position:relative;">
									<div  class="toolBarLeftName">
											<span v-if="totalChildNum == '1'"><%=rb.getString("QuanBu")%></span>
											<span v-if="totalChildNum == '2'"><%=rb.getString("HuoDongGaoJing")%></span>
											<span v-if="totalChildNum == '3'"><%=rb.getString("LiShiGaoJing")%></span>
											<el-tooltip class="item" effect="dark" :content="templateClickData.templateName" placement="top" v-if="totalChildNum == '4'&& templateClickData.length !== 0 ">
												<span v-if="totalChildNum == '4'&& templateClickData.length !== 0 " >{{templateClickData.templateName}}</span>
											</el-tooltip>
									</div>
									<div v-if="alarmViewOptShow" class="toolbarHeadBtnBoxCls">
                                        <div v-if="siteEnable" class="isFilterSiteAlarmCls">
                                            <el-checkbox v-model="params_alarm.isFilterSite" true-label="1" false-label="0"></el-checkbox>
                                            <span style="margin-left: 5px;"><%=rb.getString("GuoLvGuanLianGaoJing")%></span>
                                        </div>
										<div class="selectBlukBoxCls">
											<div class="selectMain">
												<div class="bulkSelectBtnBoxCls"  @click="openBulkSelectTable">
													<span class="el-icon-selected el-icon"></span>
													<span class="bulkSelectNumBoxCls">( {{alarmSelectData.length}} )</span>
												</div>
												<div class="selectTableBoxCls" style="position: absolute;top: 38px;left: 0px;" v-show="bulkSelectShow">
													<div class="selectBoxTitle">
														<span><%=rb.getString("YiXuan")%></span>
														<span style="position:absolute;right:20px;top:15px;" class="el-icon el-icon-close" @click="closeBulkSelectTable"></span>
													</div>
													<div class="selectBoxMain">
														<div class="tableInfoCls">
															<div class="tableInfoHeader">
																<div><%=rb.getString("PiLiangGaoJingYiXuanBiaoTi")%></div>
																<div @click="clearBulkSelected"><span style="margin-right:5px;" class="el-icon el-icon-operation-delete" ></span>Clear</div>
															</div>
															<el-ctable 
																id="bulkSelectTable" 
																ref="bulkSelectTable" 
																:data="alarmSelectData" 
																:showHeader="false"
																:rownumber="false"
																:front-pagination="true"
																:row-key="'alarm_id'"
																height="270px" pagination="true" >
																<el-table-column prop="alarm_id" v-if="false"></el-table-column>
																<el-table-column width="588">
																	<template slot-scope="scope" >
																		<div class="tableItemCls">
																			<span>{{scope.row.alarm_select_name}}</span>
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
										<div :class="alarmSelectData.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="filterBatch">
											<span class="el-icon el-icon-operation-alarm-filter"></span>
											<span><%=rb.getString("GuoLvGaoJing")%></span>
										</div>
										<div :class="alarmSelectData.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="confirmBatch">
											<span class="el-icon el-icon-operation-confirm"></span>
											<span><%=rb.getString("QueRenGaoJing")%></span>
										</div>
										<div :class="alarmSelectData.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="unConfirmBatch">
											<span class="el-icon el-icon-operation-unConfirm"></span>
											<span><%=rb.getString("FanQueRenGaoJing")%></span>
										</div>
										<div class="headBtnItemCls headBtnItemDisCls" v-show="totalChildNum != '3' && noActiveAlarmShow">
											<span class="el-icon el-icon-operation-clear"></span>
											<span><%=rb.getString("QingChuGaoJing")%></span>
										</div>
										<div :class="alarmSelectData.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="clearBatch" v-show="totalChildNum != 3 && !noActiveAlarmShow">
											<span class="el-icon el-icon-operation-clear"></span>
											<span><%=rb.getString("QingChuGaoJing")%></span>
										</div>
										<div class="headBtnItemCls headBtnItemDisCls" v-show="totalChildNum != 2 && noHistoryAlarmShow">
											<span class="el-icon el-icon-operation-delete"></span>
											<span><%=rb.getString("ShanChuGaoJing")%></span>
										</div>
										<div :class="alarmSelectData.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="deleteBatch" v-show="totalChildNum != 2 && !noHistoryAlarmShow">
											<span class="el-icon el-icon-operation-delete"></span>
											<span><%=rb.getString("ShanChuGaoJing")%></span>
										</div>
										<div :class="alarmSelectData.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="readBatch" v-show="totalChildNum != '3'">
											<span class="el-icon el-icon-read"></span>
											<span><%=rb.getString("BiaoJiWeiYiDu")%></span>
										</div>
									</div>
									<div style="position:relative;">
										<div id="tableHeadQuery" class="tableHeadQueryBoxCls specialQueryBoxCls">
                                            <div class="headQueryInputTypeBoxCls">
                                                <el-select v-model="queryInputType" @change="queryInputTypeChange" style="width: 140px;">
                                                    <el-option label='<%=rb.getString("QuanBu")%>' value="Fuzzy"></el-option>
                                                    <el-option label='<%=rb.getString("GaoJingWeiYiBiaoZhi")%>' value="Precise"></el-option>
                                                </el-select>
                                            </div>
											<div class="headQueryBox">
												<div class="queryGroup">
													<el-input v-model="search_text" @keyup.enter.native="query" @focus="queryInputFocus" @blur="queryInputBlur" :placeholder='placeholderText' style="width:260px;"></el-input>
													<i @click='query' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
												</div>
											</div>
											<div class="headTimeQueryBox">
												<div class="headTimeQueryTitleCls" style="border-right: none;">
													<%=rb.getString("GuZhangShiJian")%>
												</div>
                                                <el-date-picker
                                                    key="queryTableDateTime"
                                                    is-range
                                                    type="datetimerange" 
                                                    v-model="queryTableTime"
                                                    value-format="yyyy-MM-dd HH:mm:ss"
                                                    range-separator="—"  
                                                    @change="queryTableTimeChange"
                                                    start-placeholder='<%=rb.getString("KaiShiShiJian")%>' 
                                                    end-placeholder='<%=rb.getString("JieShuShiJian")%>'>
                                                </el-date-picker>
                                            </div>
											<div v-for="(item,index) in advancedQueryItemList">
												<div v-if="item.type == 'checkbox' && item.isShow" style="margin-right:10px;">
													<el-popfilter
														:label='item.label'
														v-model="item.checkedItemList"
														:list="item.options"
														:visible.sync="item.isShow"
														:closable="true"
														@check-change="advanceQuery(item.type,item.value,item.checkedItemList)"
														@close="checkItemDel(item)">
													</el-popfilter>
												</div>
												<div v-if="item.type == 'select' && item.isShow" style="margin-right:10px;">
													<el-popfilter
														type="single"
														:label='item.label'
														v-model="item.selectVal"
														:list="item.options"
														:visible.sync="item.isShow"
														:closable="true"
														@check-change="advanceQuery(item.type,item.value,item.selectVal)"
														@close="checkItemDel(item)">
													</el-popfilter>
												</div>
												<div v-if="item.type == 'filter'" style="margin-right:10px;">
													<div class="advancedQueryItemBox" style="background: #FFF;">
														<el-popover :ref="'popover-'+item.value" trigger="click" placement="bottom-start"  @show="checkPopoverShow(item)" @hide="checkPopoverHide(item,'')">
															<div class="checkPopoverBoxCls">
																<el-checkbox-group v-model="item.checkedItemList" @change="handleCheckedChange(item)">
																	<el-checkbox v-for=" items in item.options" :key="items.value" :label="items.value">{{items.label}}</el-checkbox>
																</el-checkbox-group>
																<div class="buttonGroup">
																	<el-button size="mini" type="primary" @click="checkPopoverSubmit(item)"><%=rb.getString("QueDing")%></el-button>
																	<el-button size="mini" @click="checkPopoverHide(item,'del')"><%=rb.getString("QuXiao")%></el-button>
																</div>	
															</div>
															<div slot="reference" class="ItemAndIconBoxCls">
																<i class="el-icon el-icon-filterAdd"></i>
																{{item.label}}
															</div>
														</el-popover>
													</div>
												</div>
											</div>
											<div class="advancedQueryItemBox"  style="background: #FFF;" @click="clearFilterClick">
												<%=rb.getString("QingKongShaiXuan")%>
											</div>
										</div>
									</div>
								</div>
							
							</template>
							<!-- 主列表 -->
							<el-table-column v-if="alarmViewOptShow" type="selection" :reserve-selection="true" ></el-table-column>
							<el-table-column label='' width="40" prop="">
								<template slot-scope="scope">
									<el-badge is-dot :hidden="scope.row.unread == '0' || !scope.row.unread">
										<div class="el-icon el-icon-operation-info grayIcon" @click="detailAlarmInfo(scope.row,event)" style="cursor: pointer;"></div>
									</el-badge>
								</template>
							</el-table-column>
							<el-table-column label='<%=rb.getString("XuHao")%>' min-width="80" prop="alarm_id" sortable></el-table-column>
							<el-table-column label='<%=rb.getString("GaoJingJiBie")%>' min-width="100"  prop="alarm_serverity_value" sortable>
								<template slot-scope="scope">
									<div v-if="scope.row.alarm_serverity_value == 'Minor'" class="alarmMinor">
										<span class="el-icon el-icon-status-alarm" style="margin-right:5px;"></span><%=rb.getString("CiYaoGaoJing")%>
									</div>
									<div v-else-if="scope.row.alarm_serverity_value == 'Major'"  class="alarmMajor">
										<span class="el-icon el-icon-status-alarm" style="margin-right:5px;"></span><%=rb.getString("ZhuYaoGaoJing")%>
									</div>
									<div v-else-if="scope.row.alarm_serverity_value == 'Critical'" class="alarmCritical">
										<span class="el-icon el-icon-status-alarm" style="margin-right:5px;"></span><%=rb.getString("JinJiGaoJing")%>
									</div>
									<div v-else-if="scope.row.alarm_serverity_value == 'Warning'" class="alarmWarning">
										<span class="el-icon el-icon-status-alarm" style="margin-right:5px;"></span><%=rb.getString("JingGaoGaoJing")%>
									</div>
								</template>
							</el-table-column>
							<el-table-column label='<%=rb.getString("GaoJingWeiYiBiaoZhi")%>' min-width="130" prop="alarm_identifier"></el-table-column>
							<el-table-column label='<%=rb.getString("KeNengYuanYin")%>' min-width="180" prop="alarm_name" show-overflow-tooltip></el-table-column>
							<el-table-column label='<%=rb.getString("XinGaoJingYuan")%>' min-width="120" prop="ne_type"></el-table-column>
							<el-table-column label='<%=rb.getString("ZhanZhiMingCheng")%>' min-width="120" prop="sub_station_name" v-if="northOperatorScenario == 'S0009'"></el-table-column>
							<el-table-column label='<%=rb.getString("WangYuanDingWei")%>' min-width="250"  prop="equip_info" show-overflow-tooltip></el-table-column>
							<el-table-column v-if="additilnalColShow" :label="siteIdLabel" prop="shop_id" min-width="80"></el-table-column>
							<el-table-column label='<%=rb.getString("ShiJianLeiXing")%>' min-width="160" prop="event_type" show-overflow-tooltip>
								<template slot-scope="scope">
									<div v-if="scope.row.event_type == '30000'">
										<span><%=rb.getString("TongXinGaoJing")%></span>
									</div>
									<div v-else-if="scope.row.event_type == '30001'">
										<span><%=rb.getString("FuWuZhiLiangGaoJing")%></span>
									</div>
									<div v-if="scope.row.event_type == '30002'">
										<span><%=rb.getString("ChuLiShiBaiGaoJing")%></span>
									</div>
									<div v-else-if="scope.row.event_type == '30003'">
										<span><%=rb.getString("SheBeiGaoJing")%></span>
									</div>
									<div v-else-if="scope.row.event_type == '30004'">
										<span><%=rb.getString("HuanJingGaoJing")%></span>
									</div>
									<div v-else-if="scope.row.event_type == '30006'">
										<span><%=rb.getString("XingNengYiChuGaoJing")%></span>
									</div>
								</template>
							</el-table-column>
							<el-table-column label='<%=rb.getString("GaoJingZhuangTai")%>' min-width="190" prop="deal_state"  show-overflow-tooltip sortable>
								<template slot-scope="scope">
									<div v-if="scope.row.deal_state == '0'" class="unconfirmInactive">
										<span class="el-icon el-icon-status-unconfirmActive" style="margin-right:5px;"></span><%=rb.getString("WeiQueRenWeiQingChu")%>
									</div>
									<div v-else-if="scope.row.deal_state == '1'" class="confirmInactive">
										<span class="el-icon el-icon-status-confirmActive" style="margin-right:5px;"></span><%=rb.getString("YiQueRenWeiQingChu")%>
									</div>
									<div v-if="scope.row.deal_state == '2'" class="unconfirmActive">
										<span class="el-icon el-icon-status-unconfirmActive" style="margin-right:5px;"></span><%=rb.getString("WeiQueRenYiQingChu")%>
									</div>
									<div v-else-if="scope.row.deal_state == '3'" class="confirmActive">
										<span class="el-icon el-icon-status-confirmActive" style="margin-right:5px;"></span><%=rb.getString("YiQueRenYiQingChu")%>
									</div>
								</template>
							</el-table-column>
							<el-table-column label='<%=rb.getString("GaoJingLeiXing")%>' min-width="100" prop="alarm_type">
								<template slot-scope="scope">
									<div v-if="scope.row.alarm_type == 'active'">
										<%=rb.getString("HuoDongGaoJing")%>
									</div>
									<div v-else-if="scope.row.alarm_type == 'history'">
										<%=rb.getString("LiShiGaoJing")%>
									</div>
								</template>
							</el-table-column>
							<el-table-column label='<%=rb.getString("GuZhangShiJian")%>' min-width="150" prop="event_time" sortable></el-table-column>
							<el-table-column label='<%=rb.getString("GengXinShiJian")%>' width="150"  prop="upd_time" sortable></el-table-column>
							<el-table-column v-if="params_alarm.alarmType == 'HISTORY'|| params_alarm.alarmType == 'ALL'" label='<%=rb.getString("GaoJingQingChuShiJian")%>' width="150"  prop="clear_time" sortable></el-table-column>
							<el-table-column label='<%=rb.getString("JuTiGuZhang")%>' min-width="150" prop="specific_problem"></el-table-column>
							<el-table-column label='<%=rb.getString("GaoJingCiShu")%>'  min-width="100" prop="alarm_count" sortable></el-table-column>
							<el-table-column label='<%=rb.getString("MiaoShu")%>'  min-width="100" prop="deal_memo"></el-table-column>

						</el-ctable>
						<el-cmenu ref="menusAlarm" :data="menusAlarm" @click="clickAlarmMenu"></el-cmenu>
						
					</div>
				</div>
			</el-tab-pane>
			<el-tab-pane v-if="alarmLibShow" label="<%=rb.getString("AlarmlevelConfig")%>" name="alarmLib">
				<!-- 操作按钮 -->
				<div class="operations">
					<div class="circleIcon placeholder-bt" placeholder="<%=rb.getString("DaoChu")%>" style="margin:0;">
						<span class="el-icon el-icon-circle-export" @click="exportAlarmLib"></span>
					</div>
				</div>
				<!-- 表格组件 -->
				<el-ctable
					:url="libraryTableUrl" 
					:query-params="libraryParams"
					id="libraryTable"
					ref="libraryTable" 
					:height="height" 
					:page-size="libraryPageSize" 
					:page-list="libraryPageList"
					@sort-change="librarySortChange" 
					pagination="true">
						<!-- 列表toolbar -->
					<template slot="toolbar">
						<el-query type="normal" @query="queryLibrary" placeholder="<%=rb.getString("GaoJingWeiYiBiaoZhi")%> / <%=rb.getString("KeNengYuanYin")%> / <%=rb.getString("XinGaoJingYuan")%>"></el-query>
					</template>
						<!-- 列表columns -->
					<el-table-column prop="DEVICE_TYPE_NAME" label="<%=rb.getString("XinGaoJingYuan")%>" min-width="110" sortable></el-table-column>
					<el-table-column prop="ALARM_IDENTIFIER" label="<%=rb.getString("GaoJingWeiYiBiaoZhi")%>" min-width="150" sortable></el-table-column>
					<el-table-column prop="ALARM_NAME" label="<%=rb.getString("KeNengYuanYin")%>" min-width="300"></el-table-column>
					<el-table-column prop="SERVERITY_TYPE" label="<%=rb.getString("YanZhongChengDu")%>" min-width="140" sortable>
						<template slot-scope="scope">
							<el-popover placement="bottom" width="150" trigger="click" popper-class="severityPopperCls">
								<div v-if="alarmViewOptShow" class="severityCls">
									<div class="alarmMinor" @click="resetAlarmServerity('31003',scope.row.ALARM_IDENTIFIER)"><span class="el-icon el-icon-status-alarm" style="margin-right:20px;"></span><%=rb.getString("CiYaoGaoJing")%></div>
									<div class="alarmMajor" @click="resetAlarmServerity('31002',scope.row.ALARM_IDENTIFIER)"><span class="el-icon el-icon-status-alarm" style="margin-right:20px;"></span><%=rb.getString("ZhuYaoGaoJing")%></div>
									<div class="alarmCritical" @click="resetAlarmServerity('31001',scope.row.ALARM_IDENTIFIER)"><span class="el-icon el-icon-status-alarm" style="margin-right:20px;"></span><%=rb.getString("JinJiGaoJing")%></div>
									<div class="alarmWarning" @click="resetAlarmServerity('31004',scope.row.ALARM_IDENTIFIER)"><span class="el-icon el-icon-status-alarm" style="margin-right:20px;"></span><%=rb.getString("JingGaoGaoJing")%></div>
								</div>
								<div v-if="scope.row.SERVERITY_TYPE == 'Minor' && alarmViewOptShow" class="alarmMinor no-outline" style="cursor: pointer;" slot="reference">
									<span class="el-icon el-icon-status-alarm" style="margin-right:5px;"></span><%=rb.getString("CiYaoGaoJing")%>
								</div>
								<div v-else-if="scope.row.SERVERITY_TYPE == 'Major' && alarmViewOptShow"  class="alarmMajor no-outline" style="cursor: pointer;" slot="reference">
									<span class="el-icon el-icon-status-alarm" style="margin-right:5px;"></span><%=rb.getString("ZhuYaoGaoJing")%>
								</div>
								<div v-else-if="scope.row.SERVERITY_TYPE == 'Critical' && alarmViewOptShow" class="alarmCritical no-outline" style="cursor: pointer;" slot="reference">
									<span class="el-icon el-icon-status-alarm" style="margin-right:5px;"></span><%=rb.getString("JinJiGaoJing")%>
								</div>
								<div v-else-if="scope.row.SERVERITY_TYPE == 'Warning' && alarmViewOptShow" class="alarmWarning no-outline" style="cursor: pointer;" slot="reference">
									<span class="el-icon el-icon-status-alarm" style="margin-right:5px;"></span><%=rb.getString("JingGaoGaoJing")%>
								</div>
							</el-popover>
							<div v-if="!alarmViewOptShow">
                                <div v-if="scope.row.SERVERITY_TYPE == 'Minor'" class="alarmMinor no-outline">
                                    <span class="el-icon el-icon-status-alarm" style="margin-right:5px;"></span><%=rb.getString("CiYaoGaoJing")%>
                                </div>
                                <div v-else-if="scope.row.SERVERITY_TYPE == 'Major'"  class="alarmMajor no-outline">
                                    <span class="el-icon el-icon-status-alarm" style="margin-right:5px;"></span><%=rb.getString("ZhuYaoGaoJing")%>
                                </div>
                                <div v-else-if="scope.row.SERVERITY_TYPE == 'Critical'" class="alarmCritical no-outline">
                                    <span class="el-icon el-icon-status-alarm" style="margin-right:5px;"></span><%=rb.getString("JinJiGaoJing")%>
                                </div>
                                <div v-else-if="scope.row.SERVERITY_TYPE == 'Warning'" class="alarmWarning no-outline">
                                    <span class="el-icon el-icon-status-alarm" style="margin-right:5px;"></span><%=rb.getString("JingGaoGaoJing")%>
                                </div>
                            </div>
						</template>
					</el-table-column>
					<el-table-column label='<%=rb.getString("ShiJianLeiXing")%>' min-width="160" prop="EVENT_TYPE" show-overflow-tooltip>
							<template slot-scope="scope">
								<div v-if="scope.row.EVENT_TYPE == '30000'">
									<span><%=rb.getString("TongXinGaoJing")%></span>
								</div>
								<div v-else-if="scope.row.EVENT_TYPE == '30001'">
									<span><%=rb.getString("FuWuZhiLiangGaoJing")%></span>
								</div>
								<div v-if="scope.row.EVENT_TYPE == '30002'">
									<span><%=rb.getString("ChuLiShiBaiGaoJing")%></span>
								</div>
								<div v-else-if="scope.row.EVENT_TYPE == '30003'">
									<span><%=rb.getString("SheBeiGaoJing")%></span>
								</div>
								<div v-else-if="scope.row.EVENT_TYPE == '30004'">
									<span><%=rb.getString("HuanJingGaoJing")%></span>
								</div>
								<div v-else-if="scope.row.EVENT_TYPE == '30006'">
									<span><%=rb.getString("XingNengYiChuGaoJing")%></span>
								</div>
							</template>
						</el-table-column>
					<el-table-column prop="EXPLANATION" label="<%=rb.getString("GaoJingJieShi")%>" min-width="700" show-overflow-tooltip></el-table-column>
				
				</el-ctable>
			</el-tab-pane>
		</el-tabs>
	</div>
	<!--  新建告警模板slide -->
    <el-slide  ref="viewAddSlide" :url='viewAddSlideUrl' :title="viewAddSlideTitle" :footer="viewAddSlideFooter" :header="viewAddSlideHeader" :position="viewAddSlidePosition"
		:height="viewAddSlideHeight" :modal='modal' :width='viewAddSlideWidth' :subloading="viewAddSubmitLoading" @ok="savaNewAlarmTmp" @cancel="cancelAddClick" :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'" >
	</el-slide>
	<!--告警邮件结果，告警详情，告警图表  slide -->
	<el-slide  ref="sharingSlide" :url='sharingSlideUrl' :title="sharingSlideTitle" :footer="sharingSlideFooter" :header="sharingSlideHeader" :position="sharingSlidePosition"
		:height="sharingSlideHeight" :modal='modal' :width='sharingSlideWidth'  @cancel="sharingSlideCancel" :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'" >
	</el-slide>
	<!--  新建告警过滤模板 -->
    <el-slide  ref="alarmFilterAddSlide" :url='alarmFilterAddUrl' :title="alarmFilterAddTitle" :footer="alarmFilterAddFooter" :header="alarmFilterAddHeader" :position="alarmFilterAddPosition"
		:height="alarmFilterAddHeight" :modal='modal' :width='alarmFilterAddWidth' :subloading="alarmFilterAddSubmitLoading" @ok="savaNewFilterTmp" @cancel="cancelFilterTmpAddClick" :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'" >
	</el-slide>
	<!--告警清除弹窗-->
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
				<el-input type="textarea" :rows="3" maxlength="500" v-model="confirmForm.description">
			</el-form-item>
		</el-form>
		<el-form v-if="clearFlag" :model="confirmForm" ref="confirmForm" label-position="top">
			<el-form-item v-if="!operationType">
				<%=rb.getString("QueRenQingChuGaoJing")%>
			</el-form-item>
			<el-form-item v-if="operationType">
				<%=rb.getString("QueRenPiLiangQingChuGaoJing")%>
				<div style="color:#999999"><%=rb.getString("PiLiangQingChuGaoJingTiShi")%></div>
			</el-form-item>
			<el-form-item label="<%=rb.getString("MiaoShu")%>" prop="description"> <!-- v-if="!confirmFlag" -->
				<el-input type="textarea" :rows="3" maxlength="500" v-model="confirmForm.description">
			</el-form-item>
		</el-form>
		<span slot="footer" class="dialog-footer">
			<div class="buttonGroup">
				<el-button type="primary" @click="confirmAlarm"><%=rb.getString("QueDing")%></el-button>
				<el-button @click="closeConfirmInfo"><%=rb.getString("QuXiao")%></el-button>
			</div>	
		</span>
	</el-dialog>
	<!--告警过滤弹窗-->
	<el-dialog title="<%=rb.getString("QueRen")%>" :visible.sync="showFilterInfo" width="500" 
		:close-on-click-modal="false" :style="styleObj" @close="showFilterInfo = false">
		<div><%=rb.getString("QueRenGuoLvGaoJing")%></div>
		<div style="font-size:12px;color:#B4B4B4"><%=rb.getString("QueRenGuoLvGaoJingTiShi")%></div>
		<span slot="footer" class="dialog-footer">
			<div class="buttonGroup">
				<el-button type="primary" @click="filterAlarm"><%=rb.getString("QueDing")%></el-button>
				<el-button @click="showFilterInfo = false"><%=rb.getString("QuXiao")%></el-button>
			</div>	
		</span>
	</el-dialog>
	<!--图表详情弹窗-->
	<el-dialog id="echartsDetail"  title='<%=rb.getString("XinXi")%>' width='80%' :visible.sync='echartLookDetail' :close-on-click-modal="false">
		<div style="height:550px;">
			<!-- 按钮  -- 详情导出 -->
			<div class="detailExport">
				<span class="el-icon el-icon-operation-export" @click="echartsDetailExport"></span>
				<div class="titleButtonText"><%=rb.getString("DaoChu")%></div>
			</div>
			<el-ctable v-if="echartLookDetail"	ref="ctableLookDetail" :url="lookDetailUrl" :query-params="params_lookDetai" :row-key="'id'"
				:height="height" :page-size="pageSize" :page-list="pageList" pagination="true" rownumber="true">
							
				<!-- 模糊查询-->
				<template slot="toolbar">
					<div class="queryGroup">
						<el-input v-model="params_lookDetai_form.searchText" @keyup.enter.native="queryAlarm" class='pairgrid-query' placeholder='<%=rb.getString("XuHao")%>/<%=rb.getString("GaoJingWeiYiBiaoZhi")%>/<%=rb.getString("KeNengYuanYin")%>/<%=rb.getString("WangYuanDingWei")%>'></el-input>
						<i @click="queryAlarm" class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
					</div>
				</template>
				<el-table-column label='<%=rb.getString("XuHao")%>' min-width="80" prop="alarm_id" sortable></el-table-column>
				<el-table-column label='<%=rb.getString("GaoJingJiBie")%>' min-width="100"  prop="alarm_serverity_value" sortable>
					<template slot-scope="scope">
						<div v-if="scope.row.alarm_serverity_value == 'Minor'" class="alarmMinor">
							<span class="el-icon el-icon-status-alarm" style="margin-right:5px;"></span><%=rb.getString("CiYaoGaoJing")%>
						</div>
						<div v-else-if="scope.row.alarm_serverity_value == 'Major'"  class="alarmMajor">
							<span class="el-icon el-icon-status-alarm" style="margin-right:5px;"></span><%=rb.getString("ZhuYaoGaoJing")%>
						</div>
						<div v-else-if="scope.row.alarm_serverity_value == 'Critical'" class="alarmCritical">
							<span class="el-icon el-icon-status-alarm" style="margin-right:5px;"></span><%=rb.getString("JinJiGaoJing")%>
						</div>
						<div v-else-if="scope.row.alarm_serverity_value == 'Warning'" class="alarmWarning">
							<span class="el-icon el-icon-status-alarm" style="margin-right:5px;"></span><%=rb.getString("JingGaoGaoJing")%>
						</div>
					</template>
				</el-table-column>
				<el-table-column label='<%=rb.getString("GaoJingWeiYiBiaoZhi")%>' min-width="130" prop="alarm_identifier" ></el-table-column>
				<el-table-column label='<%=rb.getString("KeNengYuanYin")%>' min-width="180" prop="alarm_name" show-overflow-tooltip></el-table-column>
				<el-table-column label='<%=rb.getString("XinGaoJingYuan")%>' min-width="120" prop="ne_type"></el-table-column>
				<el-table-column label='<%=rb.getString("WangYuanDingWei")%>' min-width="200"  prop="equip_info" show-overflow-tooltip></el-table-column>
				<el-table-column label='<%=rb.getString("ShiJianLeiXing")%>' min-width="150" prop="event_type" show-overflow-tooltip>
					<template slot-scope="scope">
						<div v-if="scope.row.event_type == '30000'">
							<span><%=rb.getString("TongXinGaoJing")%></span>
						</div>
						<div v-else-if="scope.row.event_type == '30001'">
							<span><%=rb.getString("FuWuZhiLiangGaoJing")%></span>
						</div>
						<div v-if="scope.row.event_type == '30002'">
							<span><%=rb.getString("ChuLiShiBaiGaoJing")%></span>
						</div>
						<div v-else-if="scope.row.event_type == '30003'">
							<span><%=rb.getString("SheBeiGaoJing")%></span>
						</div>
						<div v-else-if="scope.row.event_type == '30004'">
							<span><%=rb.getString("HuanJingGaoJing")%></span>
						</div>
						<div v-else-if="scope.row.event_type == '30006'">
							<span><%=rb.getString("XingNengYiChuGaoJing")%></span>
						</div>
					</template>
				</el-table-column>
				<el-table-column label='<%=rb.getString("GaoJingZhuangTai")%>' min-width="180" prop="deal_state"  show-overflow-tooltip>
					<template slot-scope="scope">
						<div v-if="scope.row.deal_state == '0'" class="unconfirmInactive">
							<span class="el-icon el-icon-status-unconfirmActive" style="margin-right:5px;"></span><%=rb.getString("WeiQueRenWeiQingChu")%>
						</div>
						<div v-else-if="scope.row.deal_state == '1'" class="confirmInactive">
							<span class="el-icon el-icon-status-confirmActive" style="margin-right:5px;"></span><%=rb.getString("YiQueRenWeiQingChu")%>
						</div>
						<div v-if="scope.row.deal_state == '2'" class="unconfirmActive">
							<span class="el-icon el-icon-status-unconfirmActive" style="margin-right:5px;"></span><%=rb.getString("WeiQueRenYiQingChu")%>
						</div>
						<div v-else-if="scope.row.deal_state == '3'" class="confirmActive">
							<span class="el-icon el-icon-status-confirmActive" style="margin-right:5px;"></span><%=rb.getString("YiQueRenYiQingChu")%>
						</div>
					</template>
				</el-table-column>
				<el-table-column label='<%=rb.getString("GaoJingLeiXing")%>' min-width="100" prop="alarm_type">
						<template slot-scope="scope">
							<div v-if="scope.row.alarm_type == 'active'">
								<%=rb.getString("HuoDongGaoJing")%>
							</div>
							<div v-else-if="scope.row.alarm_type == 'history'">
								<%=rb.getString("LiShiGaoJing")%>
							</div>
						</template>
					</el-table-column>
				<el-table-column label='<%=rb.getString("GuZhangShiJian")%>' min-width="150" prop="event_time" sortable></el-table-column>
				<el-table-column label='<%=rb.getString("GengXinShiJian")%>' min-width="150"  prop="upd_time" sortable></el-table-column>
				<el-table-column label='<%=rb.getString("GaoJingQingChuShiJian")%>' width="150"  prop="clear_time" sortable></el-table-column>
				<el-table-column label='<%=rb.getString("JuTiGuZhang")%>' min-width="150" prop="specific_problem"></el-table-column>
				<el-table-column label='<%=rb.getString("GaoJingCiShu")%>'  min-width="100" prop="alarm_count" sortable></el-table-column>
			</el-ctable>
		</div>
	</el-dialog>
</div>
</div>
<script>
var alarmViewVue = new Vue({
	el:"#alarmViewPage",
	 	data() {
			return {
                siteEnable: supportTopoSite,
                queryInputType: 'Fuzzy', // 查询输入框类型
				batchOperation:batchOperation,
				activeName: writableMap['CODE_ALARM_VIEW'] !== undefined?'alarmView':'alarmLib',
				confirmForm:{
					"confirmUser":'',
					"confirmTime":'',
					"description":''
				},
				styleObj:{},
				faultType:'',				
				operationType:false,         // 判断是单个操作 还是批量操作
				dialogTitle:'',              // 弹窗标题
				showConfirmInfo:false,       //显示确认告警窗口参数。 
				showFilterInfo:false,     	 // 显示过滤告警窗口参数。 
				isFilterBatch:false,         // 判断告警过滤 单一/批量
				confirmFlag:false,
				clearFlag:false,
				selectedIds:'',
				showDeviceGroupNotSelectMsg:false,
                showExportTimeNotSelectMsg:false,
				totalTriangleType:true,
				templateTriangleType:true,
				height:'100%',
				ctableTemplateUrl:"${ctx}/fault/viewConfig/queryViewConfigPageList.action",
				templatePageSize:50,
				templatePageList:[50,100,200],
				pageSize:50,
				pageList:[50,100,200],
				alarmData:[], // 告警列表数据
				menus:[], // 左侧表格点击列表
				menusTemplate:[],// 左侧菜单数据
				menusAlarm:[],// 右侧菜单数据
				rowDataTemplate:[], // 左侧菜单点击数据
				templateClickData:[],
				rowDataAlarm:[],  // 右侧告警列表点击数据
				alarmSelectData:[],
				totalChildNum:'2',
				defaultTotalChildNum:'2',
				packUpClickType:false, // 左侧列表收起，展开
				echartLookDetail:false, // 图表详情查看
				lookDetailUrl:'${ctx}/fault/view/queryViewPageList.action',
				params_lookDetai:{
					timeZone:timeZone,
					templateId:'',
					searchText:'',
					queryType:'Statistic',
					alarmType:''
				},
				params_lookDetai_form:{
					searchText:'',
				},

				sharingSlideUrl:'',
				sharingSlideTitle:'',
				sharingSlideFooter:'',
				sharingSlideHeader:'',
				sharingSlidePosition:'',
				sharingSlideHeight:'',
				sharingSlideWidth:'',

				viewAddSlideUrl:'',
				viewAddSlideTitle:'',
				viewAddSlideFooter:'',
				viewAddSlideHeader:'',
				viewAddSlidePosition:'',
				viewAddSlideHeight:'',
				viewAddSlideWidth:'',
                viewAddSubmitLoading:'',

				alarmFilterAddUrl:'',
				alarmFilterAddTitle:'',
				alarmFilterAddFooter:'',
				alarmFilterAddHeader:'',
				alarmFilterAddPosition:'',
				alarmFilterAddHeight:'',
				alarmFilterAddWidth:'',
                alarmFilterAddSubmitLoading:'',

				noticeResultSlideUrl:'',
				alarmDetailsSlideUrl:'',
				
				modal:'',
				timeZone:timeZone,
				alarmDetailUrl: '',
				params_alarm:{  //告警  -- 查询参数 
					timeZone: timeZone,
					alarmType:'ACTIVE',
					queryType:'View',
					searchText:'${search_text}',
					templateId:'',
					index:'',
					eventType:'',
					alarmServerity:'${alarm_severity}',
					alarmIdentifier:'',
					probableCause:'',
					neType:'', 
					equipInfo:'',
					alarmStatus:'',
					eventStartTime:'',
					eventEndTime:'',
					unread:'${unread}',
                    isFilterSite: '0',
                    uniqueAlarmIdentifier:'',
				},
                queryTableTime:[],
				loading:false,
				libraryParams:{
					search_text:'',
				},
				libraryTableUrl:'${ctx}/cell/fault/queryAlarmLevelInfosList.action',
				libraryPageSize:100,
				libraryPageList:[50,100,200,500],
				pickerMinDate:'',
				pickerOptions:{
				    onPick:({maxDate,minDate}) =>{
				 		this.pickerMinDate = minDate ?  minDate.getTime() : ''
				 	},
				 	disabledDate: time =>{
				 		if(this.pickerMinDate){
			     			const day1 = 3 * 24 * 3600 * 1000 - 1
				 			const maxTime = this.pickerMinDate + day1
							const minTime = this.pickerMinDate - day1
				 			return time.getTime() > maxTime || time.getTime() < minTime
						}
				 	}
				},
				exportTime:[],
				sortActive:'',
				orderActive:'',
				libSort:'',
				libOrder:'',
				bulkSelectShow:false,

				search_text:'${search_text}',
				placeholderText:'<%=rb.getString("QingShuRu")%>',
				advancedQueryItemList:[
					{
						type:'select',
						isShow:true,
						popoverShow:false,
						selectVal:'',
						label:'<%=rb.getString("YanZhongChengDu") %>',
						options:[
							{label:'<%=rb.getString("QuanBu")%>',value:''},
							{label:'<%=rb.getString("JinJiGaoJing")%>',value:'31001'},
							{label:'<%=rb.getString("ZhuYaoGaoJing")%>',value:'31002'},
							{label:'<%=rb.getString("CiYaoGaoJing")%>',value:'31003'},
							{label:'<%=rb.getString("JingGaoGaoJing")%>',value:'31004'}
						],
						value:'alarmServerity',
					},
					{
						type:'select',
						isShow:true,
						popoverShow:false,
						selectVal:'',
						label:'<%=rb.getString("ShiJianLeiXing") %>',
						options:[
							{label:'<%=rb.getString("QuanBu")%>',value:''},
							{label:'<%=rb.getString("TongXinGaoJing")%>',value:'30000'},
							{label:'<%=rb.getString("FuWuZhiLiangGaoJing")%>',value:'30001'},
							{label:'<%=rb.getString("ChuLiShiBaiGaoJing")%>',value:'30002'},
							{label:'<%=rb.getString("SheBeiGaoJing")%>',value:'30003'},
							{label:'<%=rb.getString("HuanJingGaoJing")%>',value:'30004'},
						],
						value:'eventType',
					},
					{
						type:'select',
						isShow:true,
						popoverShow:false,
						selectVal:'',
						label:'<%=rb.getString("XinGaoJingYuan") %>',
						options:[],
						value:'neType',
					},
					{
						type:'select',
						isShow:true,
						popoverShow:false,
						selectVal:'',
						label:'<%=rb.getString("DuYueZhuangTai") %>',
						options:[
							{label:'<%=rb.getString("QuanBu")%>',value:''},
							{label:'<%=rb.getString("YiDuYue")%>',value:'0'},
							{label:'<%=rb.getString("WeiDuYue")%>',value:'1'},
						],
						value:'unread',
					},
					{
						type:'select',
						isShow:false,
						popoverShow:false,
						selectVal:'',
						label:'<%=rb.getString("GaoJingZhuangTai") %>',
						options:[
							{label:'<%=rb.getString("QuanBu")%>',value:''},
							{label:'<%=rb.getString("WeiQueRenWeiQingChu")%>',value:'0'},
							{label:'<%=rb.getString("YiQueRenWeiQingChu")%>',value:'1'},
						],
						value:'alarmStatus',
					},
					{
						type:'filter',
						popoverShow:false,
						checkedItemList:['alarmServerity','eventType','neType','unread'],
						oldCheckedItemList:['alarmServerity','eventType','neType','unread'],
						label:'<%=rb.getString("TianJiaShuaiXuan") %>',
						options:[
							{label:'<%=rb.getString("YanZhongChengDu") %>',value:"alarmServerity"},
							{label:'<%=rb.getString("ShiJianLeiXing")%>',value:"eventType"},
							{label:'<%=rb.getString("XinGaoJingYuan")%>',value:"neType"},
							{label:'<%=rb.getString("DuYueZhuangTai")%>',value:"unread"},
							{label:'<%=rb.getString("GaoJingZhuangTai")%>',value:"alarmStatus"},
						],
						value:'add_filter',
					}

				],
			};
	  	},
		beforeUpdate(){
			this.$nextTick(()=>{
				this.$refs.ctableAlarm.doLayout();
			})
		},
		computed: {
			limitAlarm(){
				var limitNum;
				limitNum = this.batchOperation ? 100 : 1;
				return limitNum;
			},
			alarmViewOptShow() {
				return writableMap['CODE_ALARM_VIEW'] == true;
			},
			alarmViewTabShow() {
				return writableMap['CODE_ALARM_VIEW'] !== undefined;
			},
			alarmLibShow() {
				return writableMap['CODE_ALARM_LIBRARY'] !== undefined;
			},
			noHistoryAlarmShow(){
				var alarmSelectData = this.alarmSelectData,
					result = alarmSelectData.some(item=>item.alarm_type == 'history');
				
				return result ? false : true;
			},
			noActiveAlarmShow(){
				var alarmSelectData = this.alarmSelectData,
					result = alarmSelectData.some(item=>item.alarm_type == 'active');
				
				return result ? false : true;
			},
			additilnalColShow(){
				return enbAdditionalColShow == 'true' ? true : false;
			},
			siteIdLabel(){
				return siteIdLabel
			},
			siteNameLabel(){
				return siteNameLabel
			},
		},
		watch:{
			alarmSelectData(newVal){
				if(newVal.length == 0){
					this.bulkSelectShow = false;
				}
			},
		},
		methods:{
			// 打开已选弹窗
			openBulkSelectTable(){
				var vm = this;
				vm.bulkSelectShow = true
			},
			// 关闭已选弹窗
			closeBulkSelectTable(){
				var vm = this;
				vm.bulkSelectShow = false;
			},
			// 设备已选表格 清空事件
			clearBulkSelected(){
				var vm = this;

				vm.$refs.ctableAlarm.clearSelection();
			},
			// 设备已选表格 单个删除事件
			delBulkSelected(rows){
				var vm = this,
					tabs = 'ctableAlarm',
					rowKey = 'alarm_id';
				vm.alarmSelectData = vm.alarmSelectData.filter((items)=>{
					return items[rowKey] != rows[rowKey]
				});
				var selection = this.$refs[tabs].$refs.ctableInner.store.states.selection,
					irow= selection.filter((items)=>{
						return items[rowKey] == rows[rowKey]
					})[0];
				vm.$refs[tabs].toggleRowSelection(irow,false);
				var idx = vm.$refs[tabs].ckList.indexOf(rows[rowKey]);
				vm.$refs[tabs].ckList.splice(idx,1);
			},
			// tab 切换事件
			tabClick(){
				var vm = this;
				this.$nextTick(function(){
					document.body.click();
				});
				vm.$refs.ctableAlarm.clearSelection();
			},
			// 左侧总量子菜单点击事件
			totalClick(val){
					var vm = this;
					vm.params_alarm.templateId = '';
					if(val == '1'){
						vm.params_alarm.alarmType = 'ALL';
						vm.advancedQueryItemList.map((items)=>{
							if('alarmStatus' == items.value){
								items.options = [  //告警 -- 告警状态下拉选择数据 
									{label:'<%=rb.getString("QuanBu")%>',id:''},
									{label:'<%=rb.getString("WeiQueRenWeiQingChu")%>',value:'0'},
									{label:'<%=rb.getString("YiQueRenWeiQingChu")%>',value:'1'},
									{label:'<%=rb.getString("WeiQueRenYiQingChu")%>',value:'2'},
									{label:'<%=rb.getString("YiQueRenYiQingChu")%>',value:'3'},
								];
							}
                            if('unread' == items.value){
                                items.isShow = true;
                            }
                            if('add_filter' == items.value){
                                items.checkedItemList = ['alarmServerity','eventType','neType','unread'],
                                items.oldCheckedItemList = ['alarmServerity','eventType','neType','unread'],
                                items.options = [
                                    {label:'<%=rb.getString("YanZhongChengDu") %>',value:"alarmServerity"},
                                    {label:'<%=rb.getString("ShiJianLeiXing")%>',value:"eventType"},
                                    {label:'<%=rb.getString("XinGaoJingYuan")%>',value:"neType"},
                                    {label:'<%=rb.getString("DuYueZhuangTai")%>',value:"unread"},
                                    {label:'<%=rb.getString("GaoJingZhuangTai")%>',value:"alarmStatus"},
                                ];
                            }
						})
					}else if(val == '2'){
						vm.params_alarm.alarmType = 'ACTIVE';
						vm.advancedQueryItemList.map((items)=>{
							if('alarmStatus' == items.value){
								items.options = [  //告警 -- 告警状态下拉选择数据 
									{label:'<%=rb.getString("QuanBu")%>',id:''},
									{label:'<%=rb.getString("WeiQueRenWeiQingChu")%>',value:'0'},
									{label:'<%=rb.getString("YiQueRenWeiQingChu")%>',value:'1'},
								];
							}
                            if('unread' == items.value){
                                items.isShow = true;
                            }
                            if('add_filter' == items.value){
                                items.checkedItemList = ['alarmServerity','eventType','neType','unread'],
                                items.oldCheckedItemList = ['alarmServerity','eventType','neType','unread'],
                                items.options = [
                                    {label:'<%=rb.getString("YanZhongChengDu") %>',value:"alarmServerity"},
                                    {label:'<%=rb.getString("ShiJianLeiXing")%>',value:"eventType"},
                                    {label:'<%=rb.getString("XinGaoJingYuan")%>',value:"neType"},
                                    {label:'<%=rb.getString("DuYueZhuangTai")%>',value:"unread"},
                                    {label:'<%=rb.getString("GaoJingZhuangTai")%>',value:"alarmStatus"},
                                ];
                            }
						})
					}else if(val == '3'){
						vm.params_alarm.alarmType = 'HISTORY';
						vm.advancedQueryItemList.map((items)=>{
							if('alarmStatus' == items.value){
								items.options = [  //告警 -- 告警状态下拉选择数据 
									{label:'<%=rb.getString("QuanBu")%>',id:''},
									{label:'<%=rb.getString("WeiQueRenYiQingChu")%>',value:'2'},
									{label:'<%=rb.getString("YiQueRenYiQingChu")%>',value:'3'},
								];
							}
                            if('unread' == items.value){
                                items.isShow = false;
                            }
                            if('add_filter' == items.value){
                                items.checkedItemList = ['alarmServerity','eventType','neType'],
                                items.oldCheckedItemList = ['alarmServerity','eventType','neType'],
                                items.options = [
                                    {label:'<%=rb.getString("YanZhongChengDu") %>',value:"alarmServerity"},
                                    {label:'<%=rb.getString("ShiJianLeiXing")%>',value:"eventType"},
                                    {label:'<%=rb.getString("XinGaoJingYuan")%>',value:"neType"},
                                    {label:'<%=rb.getString("GaoJingZhuangTai")%>',value:"alarmStatus"},
                                ];
                            }
						})
					}
					if(vm.defaultTotalChildNum !== val){
						vm.loading = true;
						vm.$refs.ctableAlarm.clearSelection();
					}
					vm.defaultTotalChildNum = val;
					vm.alarmQuery('');
					if(val !== '4'){
						vm.$refs.ctableTemplate.setCurrentRow();
						
					}
					
			},
			// 新增告警模板
			addTemplate(){
					var vm = this;
					vm.viewAddSlideHeader = true;
					vm.viewAddSlideUrl = '${ctx}/fault/viewConfig/goAddViewConfigPage.action';
					vm.viewAddSlideFooter = true;
					vm.viewAddSlidePosition = 'top';
					vm.viewAddSlideHeight = '100%';
					vm.viewAddSlideWidth = '100%';
                    vm.viewAddSubmitLoading = false;
					vm.viewAddSlideTitle = '<%=rb.getString("XinJianGaoJingMoBan")%>';
					vm.$refs.viewAddSlide.showSlide(()=>{
							vm.model = false;
							eventBus.$emit('action-init','',timeZone,'add');
					})
					event.stopPropagation();// 禁止事件穿透
			},
			//新增视图模板 保存
			savaNewAlarmTmp(){
				var vm = this;
				eventBus.$emit("handle-ok-new");
			},
			// 打开告警过滤新建页面
			openAddFilterTmp(type,ruleId){
				var vm = this,
					faultType = type,
					timeZone = vm.timeZone;
				if(faultType == 'add'){
					vm.alarmFilterAddTitle = '<%=rb.getString("XinJianGaoJingGuoLvMoBan")%>';
					vm.alarmFilterAddFooter = true;
				}else if(faultType == 'view'){
					vm.alarmFilterAddTitle = '<%=rb.getString("XinXi")%>';
					vm.alarmFilterAddFooter = false;
				}else if(faultType == 'edit'){
					vm.alarmFilterAddTitle = '<%=rb.getString("XiuGai")%>';
					vm.alarmFilterAddFooter = true;
				}
				vm.alarmFilterAddHeader = true;
				vm.alarmFilterAddUrl = '${ctx}/cell/fault/goAddAlarmRule.action';
				vm.alarmFilterAddPosition = 'top';
				vm.alarmFilterAddHeight = '100%';
				vm.alarmFilterAddWidth = '100%';
                vm.alarmFilterAddSubmitLoading = false;
				vm.$refs.alarmFilterAddSlide.showSlide(()=>{
					if(faultType == 'view'){
						eventBus.$emit('rule-info-task',ruleId,timeZone,'view')
					}else if(faultType == 'edit'){
						eventBus.$emit('rule-info-task',ruleId,timeZone,'edit')
					}else if(faultType == 'add'){
						eventBus.$emit('rule-info-task',ruleId,timeZone,'add')
					}
				})
			},
			//新增过滤模板 保存
			savaNewFilterTmp(){
				var vm = this;
				eventBus.$emit("filter-ok-new");
			},
			// 关闭视图新建页面
			cancelAdd(){ // 关闭新建页面
				var vm = this;
				vm.$refs.viewAddSlide.hide();
				vm.$refs.ctableTemplate.refresh();
				vm.$refs.ctableAlarm.refresh();//表格刷新
			},
			// 点击视图新建页面关闭按钮
			cancelAddClick(){ // 关闭新建页面
				var vm = this;
				eventBus.$emit('alarm-add-cancel')
			},
			// 关闭过滤新建页面
			cancelFilterTmpAdd(){ 
				var vm = this;
				vm.$refs.alarmFilterAddSlide.hide();
			},
			// 点击过滤新建页面关闭按钮
			cancelFilterTmpAddClick(){ // 关闭新建页面
				var vm = this;
				eventBus.$emit('filter-add-cancel');
			},
			/**
			*  左模板列表选中事件
			* @param currentRow{object}   已选中行数据
			* @param oldCurrentRow{object}   上一个已选中行数据
			*/ 
			selectTemplate(currentRow,oldCurrentRow){
				var vm = this;

				if(currentRow !== undefined){
					vm.templateClickData = currentRow;
					vm.advancedQueryItemList.map((items)=>{
						if('alarmStatus' == items.value){
							items.options = [  //告警 -- 告警状态下拉选择数据 
								{label:'<%=rb.getString("QuanBu")%>',id:''},
								{label:'<%=rb.getString("WeiQueRenWeiQingChu")%>',value:'0'},
								{label:'<%=rb.getString("YiQueRenWeiQingChu")%>',value:'1'},
								{label:'<%=rb.getString("WeiQueRenYiQingChu")%>',value:'2'},
								{label:'<%=rb.getString("YiQueRenYiQingChu")%>',value:'3'},
							];
						}
                        if('unread' == items.value){
                            items.isShow = false;
                        }
                        if('add_filter' == items.value){
                            items.checkedItemList = ['alarmServerity','eventType','neType'],
                            items.oldCheckedItemList = ['alarmServerity','eventType','neType'],
                            items.options = [
                                {label:'<%=rb.getString("YanZhongChengDu") %>',value:"alarmServerity"},
                                {label:'<%=rb.getString("ShiJianLeiXing")%>',value:"eventType"},
                                {label:'<%=rb.getString("XinGaoJingYuan")%>',value:"neType"},
                                {label:'<%=rb.getString("DuYueZhuangTai")%>',value:"unread"},
                                {label:'<%=rb.getString("GaoJingZhuangTai")%>',value:"alarmStatus"},
                            ];
                        }
					})
					if(vm.params_alarm.templateId != currentRow.templateId){
						vm.$refs.ctableAlarm.clearSelection();
						vm.loading = true;
					}else{
						vm.$refs.ctableAlarm.refresh();
					}
					vm.params_alarm.templateId = currentRow.templateId;
					vm.params_alarm.alarmType ='ALL';
					vm.totalChildNum = '4';
					vm.alarmQuery('');
					
				}else{
					vm.templateClickData = [];
				}
			
			},
			/**
			* 左侧模板列表 点击更多操作出现菜单
			* @param row{object}   行数据
			* @param ev{object}   event数据
			*/ 
			templateOptClick(row,ev){ // 操作项： 1.结果  2.详情  3.修改  4.删除  
					var vm = this,disableFlag = "";
					vm.rowDataTemplate = row;
					if(row.edit == true){
						disableFlag = false;
					}else{
						disableFlag = true;
					} 
					vm.menusTemplate= [
								{label:'<%=rb.getString("MoBanYouJianJieGuo")%>',cls:"el-icon el-icon-operation-mail",code:'noticeResult'},
								{label:'<%=rb.getString("XinXi")%>',cls:"el-icon el-icon-operation-info",code:'info'},
								{label:'<%=rb.getString("XiuGai")%>',cls:"el-icon el-icon-operation-edit CODE_ALARM_VIEW hidden",disable:disableFlag,code:'modify',},
								{label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete CODE_ALARM_VIEW hidden",disable:disableFlag,code:'del',}
					]
					
					vm.$nextTick(function(){
							document.body.click();
							vm.$refs.menusTemplate.show(ev);
					});
					event.stopPropagation();
			},
			/**
			* 左侧菜单点击事件
			* @param ev{object}   行数据
			*/ 
			clickTemplateMenu(ev){ //单点击方法 -- 设备组列表操作  
					var vm = this;
					var codes = {
						noticeResult:vm.viewNoticeResult,
						info:vm.viewTemplateInfo,
						modify:vm.editAlarmTemp,
						del:vm.delAlarmTemp,
					}
					if(codes[ev.code]){
						codes[ev.code](vm.rowDataTemplate.templateId)
					}
			},
			//点击页面其他地方菜单收起
			handerClose(){ 
				this.$refs.menusTemplate.hide();
				this.$refs.menusAlarm.hide();
			},
			/**
			* 打开告警模板通知结果页面
			* @param row{object}   行数据
			*/
			viewNoticeResult(templateId){
				var vm = this;
				
				vm.sharingSlideUrl = "${ctx}/cell/fault/goEmailResult.action";
				vm.sharingSlideHeight = '100%';
				vm.sharingSlideWidth = '100%';
				vm.sharingSlidePosition = 'top';
				vm.sharingSlideFooter = false;
				vm.sharingSlideHeader = true;
				vm.sharingSlideTitle = '<%=rb.getString("MoBanYouJianJieGuo")%>';
				vm.$refs.sharingSlide.showSlide(function(){
					vm.modal = false;
					eventBus.$emit('result-info',templateId);
				});
			},
			// 关闭告警模板通知结果页面
			sharingSlideCancel(){
				this.$refs.sharingSlide.hide();
				this.$refs.ctableAlarm.refresh();
			},
			/**
			* 模板查看页面
			* @param templateId{number}   模板id
			*/
			viewTemplateInfo(templateId){
				var vm = this;
				vm.viewAddSlideUrl = '${ctx}/fault/viewConfig/goAddViewConfigPage.action?type=view';
				vm.viewAddSlideTitle = '<%=rb.getString("XinXi")%>';
				vm.viewAddSlidePosition = 'top';
				vm.viewAddSlideHeight = '100%';
				vm.viewAddSlideWidth = '100%';
				vm.viewAddSlideFooter = false;
				vm.viewAddSlideHeader = true;
                vm.viewAddSubmitLoading = false;
				vm.$refs.viewAddSlide.showSlide(function(){
					vm.modal = false;
					eventBus.$emit('action-init',templateId,timeZone,'view');
				});
			},
			/**
			* 模板修改页面
			* @param templateId{number}   模板id
			*/
			editAlarmTemp(templateId){
				var vm = this;
				vm.viewAddSlideUrl = '${ctx}/fault/viewConfig/goAddViewConfigPage.action?type=edit';
				vm.viewAddSlideTitle = '<%=rb.getString("XiuGai")%>';
				vm.viewAddSlidePosition = 'top';
				vm.viewAddSlideHeight = '100%';
				vm.viewAddSlideWidth = '100%';
				vm.viewAddSlideFooter = true;
				vm.viewAddSlideHeader = true;
                vm.viewAddSubmitLoading = false;
				vm.$refs.viewAddSlide.showSlide(function(){
					eventBus.$emit('action-init',templateId,timeZone,'edit');
				});
			},
			/**
			* 模板删除事件
			* @param templateId{number}    模板id
			*/
			delAlarmTemp(templateId) {
				var vm = this,
					params = {
						templateId: templateId
					};
				vm.$confirm('<%=rb.getString("QueRenShanChuMuBan")%>','<%=rb.getString("QueRen")%>').then(function(){

					axios.post('${ctx}/fault/viewConfig/deleteViewConfig.action',stringify(params)).then(function(response){
						var data = response.data;
						if(data) {
							if(data["success"]){
								vm.$message({
									message: '<%=rb.getString("ChengGong")%>',
									type:'success'
								});
								vm.totalChildNum = '1';
								vm.totalClick('1');
							}else{
								vm.$message.error(data["message"])
							}
							vm.$refs.ctableTemplate.refresh()
						}
					}).catch(function(error){})
				});
			},
			// 前往告警通知设置页面
			goAlarmNoticePage(){
				var vm = this;
				vm.sharingSlideUrl = '${ctx}/fault/viewConfig/goGlobalViewConfigPage.action';
				vm.sharingSlideHeight = '100%';
				vm.sharingSlideWidth = '100%';
				vm.sharingSlideFooter = false;
				vm.sharingSlidePosition = 'right';
				vm.sharingSlideHeader = true;
				vm.sharingSlideTitle = '<%=rb.getString("TongZhiSheZhi")%> ';
				vm.$refs.sharingSlide.showSlide(function(){
					vm.modal = false;
				});
			},
			// 前往告警过滤页面
			goFilterPage(){
				var vm = this;
				vm.sharingSlideUrl = '${ctx}/cell/fault/goAlarmRule.action';
				vm.sharingSlideHeight = '100%';
				vm.sharingSlideWidth = '100%';
				vm.sharingSlideFooter = false;
				vm.sharingSlidePosition = 'right';
				vm.sharingSlideHeader = true;
				vm.sharingSlideTitle = '<%=rb.getString("GaoJingGuoLv")%>';
				vm.$refs.sharingSlide.showSlide(function(){
					vm.modal = false;
				});
			},
			// 前往告警图表页面
			goAlarmCharts(){
				var vm = this;
				vm.sharingSlideUrl = '${ctx}/cell/fault/goStatisticTotalPage.action';
				vm.sharingSlideHeight = '100%';
				vm.sharingSlideWidth = '100%';
				vm.sharingSlideFooter = false;
				vm.sharingSlidePosition = 'top';
				vm.sharingSlideHeader = false;
				vm.sharingSlideTitle = '<%=rb.getString("GaoJingTuBiao")%>';
				vm.$refs.sharingSlide.showSlide(function(){
					vm.modal = false;
				});
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
				vm.showDeviceGroupNotSelectMsg = false;
			},
            // 导出时间选择
            exportTimeChange(val){
                var vm = this;
                if(val && val.length > 0){
                    vm.showExportTimeNotSelectMsg = false;
                }else{
                    vm.showExportTimeNotSelectMsg = true;
                    vm.pickerMinDate = '';
                }
            },
			// 导出数据
			openExportDialg(){
				var vm = this,
                    urls ='${ctx}/fault/view/exportViewPageList.action',
                    params={
                        timeZone: timeZone,
                        searchText: vm.params_alarm.searchText,
                        alarmType: vm.params_alarm.alarmType,
                        queryType: vm.params_alarm.queryType,
                        index: vm.params_alarm.index,
                        templateId: vm.params_alarm.templateId,
                        alarmServerity: vm.params_alarm.alarmServerity,
                        alarmIdentifier: vm.params_alarm.alarmIdentifier,
                        eventType: vm.params_alarm.eventType,
                        probableCause: vm.params_alarm.probableCause,
                        neType: vm.params_alarm.neType,
                        equipInfo: vm.params_alarm.equipInfo,
                        alarmStatus: vm.params_alarm.alarmStatus,
                        unread: vm.params_alarm.unread,
                        eventStartTime: vm.params_alarm.eventStartTime,
                        eventEndTime: vm.params_alarm.eventEndTime,
                        uniqueAlarmIdentifier: vm.params_alarm.uniqueAlarmIdentifier,
                        isFilterSite: vm.params_alarm.isFilterSite,
                        sort: vm.sortActive,
                        order: vm.orderActive,
                    };
				if(vm.totalChildNum !== '4'){
                    let newDayTime = formatDate(new Date(gloableTime)).slice(0,10);
                    vm.exportTime = [newDayTime + ' 00:00:00',newDayTime + ' 23:59:59'];
                    vm.showExportTimeNotSelectMsg = false;
				}else{
					exportByForm(urls,params);
				}  
			},
			// 导出确认
			exportAlarm(){ 
				var vm = this,
                    urls ='${ctx}/fault/view/exportViewPageList.action',
					params={
                        timeZone: timeZone,
                        searchText: vm.params_alarm.searchText,
                        alarmType: vm.params_alarm.alarmType,
                        queryType: vm.params_alarm.queryType,
                        index: vm.params_alarm.index,
                        templateId: vm.params_alarm.templateId,
                        alarmServerity: vm.params_alarm.alarmServerity,
                        alarmIdentifier: vm.params_alarm.alarmIdentifier,
                        eventType: vm.params_alarm.eventType,
                        probableCause: vm.params_alarm.probableCause,
                        neType: vm.params_alarm.neType,
                        equipInfo: vm.params_alarm.equipInfo,
                        alarmStatus: vm.params_alarm.alarmStatus,
                        unread: vm.params_alarm.unread,
                        uniqueAlarmIdentifier: vm.params_alarm.uniqueAlarmIdentifier,
                        isFilterSite: vm.params_alarm.isFilterSite,
                        eventStartTime: '',
                        eventEndTime: '',
                        deviceGroup: '',
                        sort: vm.sortActive,
                        order: vm.orderActive,
                    };
				var selection = vm.$refs.ctableDeviceGroup.getChecked();
				if(selection.length == 0){
					vm.showDeviceGroupNotSelectMsg = true;
                    return
				}else{
                    params.deviceGroup = (vm.selectedIds).join(",");
					vm.showDeviceGroupNotSelectMsg = false;
				}
                if(vm.exportTime && vm.exportTime.length > 0){
                    vm.showExportTimeNotSelectMsg = true;
                    params.eventStartTime = vm.exportTime[0];
                    params.eventEndTime = vm.exportTime[1];
                }else{
                    vm.showExportTimeNotSelectMsg = true;
                    return
                }
                exportByForm(urls,params);
                vm.closeExport();
			},
			// 取消导出
			closeExport(){ 
                this.exportTime = [];
				this.$refs.ctableDeviceGroup.clearSelection();
				this.showDeviceGroupNotSelectMsg = false;
                this.showExportTimeNotSelectMsg = false;
                document.body.click();
			},
			/**
			* 基站设备列表 点击更多操作出现菜单
			* @param row{object}   行数据
			* @param ev{object}   event数据
			*/ 
			optAlarmClick(row,ev){ // 操作项 
				var vm = this ,
					showClear = true,
					confirmDis = true; 
				vm.rowDataAlarm = row;
				vm.faultType = row.alarm_type;
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
					{label:'<%=rb.getString("GuoLvGaoJing")%>',cls:"el-icon-operation-alarm-filter el-icon CODE_ALARM_VIEW hidden",code:'filter'},
					{label:'<%=rb.getString("QueRenGaoJing")%>',cls:"el-icon-operation-confirm el-icon CODE_ALARM_VIEW hidden" ,code:'confirm'},
					{label:'<%=rb.getString("FanQueRenGaoJing")%>',cls:"el-icon-operation-unconfirm el-icon CODE_ALARM_VIEW hidden" ,code:'unConfirm',disable:!confirmDis},
					{label:'<%=rb.getString("QingChuGaoJing")%>',cls:"el-icon-operation-clear el-icon CODE_ALARM_VIEW hidden",code:'clear',show:showClear},
					{label:'<%=rb.getString("ShanChuGaoJing")%>',cls:"el-icon-operation-delete el-icon CODE_ALARM_VIEW hidden",code:'del',show:!showClear}
				];
				
				vm.$nextTick(function(){
					document.body.click();
					vm.$refs.menusAlarm.show(ev);
				});
			},
			/**
			* 右侧菜单点击事件
			* @param ev{object}   行数据
			*/ 
			clickAlarmMenu(ev){ //单点击方法 -- 设备组列表操作  
				var vm = this ; 
				var codes = {
					detail:vm.detailAlarmInfo,
					filter:vm.openFilterDialog,
					confirm:vm.openConfirmDialog,
					unConfirm:vm.unconfirmAlarm,
					clear:vm.clearAlarm,
					del:vm.deleteAlarm
				}
				if(codes[ev.code]){
					codes[ev.code](vm.rowDataAlarm)
				}
			},
			/**
			* 打开告警详情页面
			* @param row{object}   行数据
			*/  
			detailAlarmInfo(row,ev){
				var vm = this;
					
				vm.sharingSlideUrl = "${ctx}/cell/fault/goAlarmDetail.action";
				vm.sharingSlideHeight = '100%';
				vm.sharingSlideWidth = '100%';
				vm.sharingSlideFooter = false;
				vm.sharingSlidePosition = 'top';
				vm.sharingSlideHeader = true;
				vm.sharingSlideTitle = '<%=rb.getString("XiangQing")%>';
				vm.$refs.sharingSlide.showSlide(function(){
					vm.modal = false;
					eventBus.$emit('detail-info',row.alarm_type,row.alarm_id);
				});
			},
			// 关闭告警详情页面
			cancelAlarmDetails(){
				this.$refs.viewAlarmDetailsSlide.hide();
			},
			// 打开告警过滤弹窗
			openFilterDialog(){
				this.showFilterInfo = true;
				this.isFilterBatch = false;
				this.styleObj = {
						'marginTop':'20vh'
					}
			},
			// 过滤告警确认
			filterAlarm(){
				var vm = this,
					alarmIds=[];
				if(!vm.isFilterBatch){
					var params = {};
					params.alarm_id = vm.rowDataAlarm.alarm_identifier;
					alarmIds.push(params);
				}else{
					vm.alarmSelectData.map((item,index) => {
						alarmIds.push({alarm_id:item.alarm_identifier})
						return alarmIds;
					})
				}
				let alarmIdsData = JSON.stringify(alarmIds);
				axios.post('${ctx}/cell/fault/batchFilterAlarm.action',alarmIdsData,{headers:{'Content-Type':'application/json;charset=utf-8'},}).then(function(response){
					var data = response.data;
					if(data["success"]){
						vm.$message.success('<%=rb.getString("ChengGong")%>')
						vm.$refs.ctableAlarm.clearSelection();
						vm.$refs.ctableAlarm.refresh();//表格刷新
						vm.showFilterInfo = false;
					}else{
						vm.$message.error('<%=rb.getString("GuoLvGaoJingShiBai")%>') //错误提示信息 
					}
				}) 
			},
			/**
			* 打开告警确认窗口 
			* @param row{object}   行数据
			*/  
			openConfirmDialog(row){
				var vm = this,type= vm.faultType,
					status = row.deal_state;
				
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
					alarm_id : vm.rowDataAlarm.alarm_id,
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
					type=vm.faultType,
					msg;
				if(!vm.operationType){
					if( vm.clearFlag == true){
						url = '${ctx}/cell/fault/clearAlarm.action';    //清除告警 
						msg = '<%=rb.getString("QingChuGaoJingTiShi")%>'
					}else {
						url = '${ctx}/cell/fault/confirmAlarm.action';     //确认告警 
						msg = '<%=rb.getString("QueRenGaoJingTiShi")%>'
					}
					
					axios.post(url,stringify({
						alarm_id : vm.rowDataAlarm.alarm_id,
						small_cell_code: vm.rowDataAlarm.small_cell_code,
						type : type,
						text : vm.confirmForm.description
					})).then(function(response){
						var data = response.data;
						if(data["success"]){
							vm.closeConfirmInfo();
							if(vm.clearFlag){
								vm.alarmSelectData.map((item)=>{
									if(item.alarm_id == vm.rowDataAlarm.alarm_id){
										vm.$refs.ctableAlarm.toggleRowSelection(item,false)
									}
								})
							}
							vm.$message.success( '<%=rb.getString("ChengGong")%>');
							vm.$refs.ctableAlarm.refresh();//表格刷新
						}else{
							vm.closeConfirmInfo();
							vm.$message.error(msg + '<%=rb.getString("TiShiShiBai")%>') //错误提示信息
						}
					}) 
				}else{
					var	alarmIds=[];
					if(vm.clearFlag == true){
						vm.alarmSelectData.map((item,index) => {
							if(item.alarm_type == 'active'){
								alarmIds.push({alarm_id:item.alarm_id,small_cell_code:item.small_cell_code,type:item.alarm_type,text:vm.confirmForm.description,alarm_identifier:item.alarm_identifier})
							}
							return alarmIds;
						})
					}else{
						vm.alarmSelectData.map((item,index) => {
							alarmIds.push({alarm_id:item.alarm_id,small_cell_code:item.small_cell_code,type:item.alarm_type,text:vm.confirmForm.description})
							return alarmIds;
						})
					}
						
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
							vm.$refs.ctableAlarm.clearSelection();
							vm.$message.success( '<%=rb.getString("ChengGong")%>');
							vm.$refs.ctableAlarm.refresh();//表格刷新
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
					type= vm.faultType;
				vm.operationType = false;
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
							vm.$refs.ctableAlarm.refresh();//表格刷新
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
				var vm = this,type= vm.faultType,
					status = row.deal_state;
				
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
				axios.post('${ctx}/cell/fault/queryAlarmDetail.action',stringify({
					alarm_id : vm.rowDataAlarm.alarm_id,
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
						small_cell_code:vm.rowDataAlarm.small_cell_code,
					})).then(function(response){
						var data = response.data;
						if(data["success"]){
							vm.$message.success('<%=rb.getString("ChengGong")%>')
							vm.$refs.ctableAlarm.refresh();//表格刷新
							vm.$refs.ctableAlarm.toggleRowSelection(row,false);
						}else{
							vm.$message.error('<%=rb.getString("ShanChuGaoJingShiBai")%>') //错误提示信息 
						}
					}) 
				}).catch(function(){
					
				})
				
			},
			// 打开批量过滤弹窗
			filterBatch(){
				var vm = this;

				if(vm.alarmSelectData.length <= 0)return

			vm.showFilterInfo = true;
			vm.isFilterBatch = true;
			vm.styleObj = {
						'marginTop':'20vh'
				}
		},
		//批量标记为已读
		readBatch(){
			var vm = this,params={};
			if(vm.alarmSelectData.length <= 0)return
			var alarmIds='';
			vm.alarmSelectData.map((item,index) => {
				if(item.unread && item.unread == '1'){
					alarmIds += item.alarm_id + ',';
				}
				return alarmIds;
			})
			alarmIds = alarmIds.substring(0,alarmIds.length-1);
			//如果没有未读的告警则不请求后台，并且给出提示
			if(alarmIds == '') {
				vm.$message({
					message: '<%=rb.getString("MeiYouWeiDuGaoJing")%>',
					type: 'warning'
				});
				return;
			}
			params.alarmIds = alarmIds;
			axios.post('${ctx}/fault/viewConfig/updateUnreadAlarmAlert.action',stringify(params)).then(function(response){
				var data = response.data;
				if(data) {
					if(data["success"]){
						vm.$refs.ctableAlarm.clearSelection();
						vm.$refs.ctableAlarm.refresh();//表格刷新
						
					}else{
						//如果消息为空，则提示失败
						if(data["message"] == null || data["message"] == ''){
							vm.$message.error('<%=rb.getString("ShiBai")%>')
						}else{
							vm.$message.error(data["message"])
						}
					}
				}
			}).catch(function(error){})
		},
		//批量删除
		deleteBatch(){
			var vm = this;

				if(vm.alarmSelectData.length <= 0)return
				var	alarmIds=[];
				vm.alarmSelectData.map((item,index) => {
					if(item.alarm_type == 'history'){
						alarmIds.push({alarm_id:item.alarm_id,small_cell_code:item.small_cell_code})
					}
					return alarmIds;
				})
				let alarmIdsData = JSON.stringify(alarmIds);
				var confirmHint ='<div style="font-size:14px;color:#333333">'+ '<%=rb.getString("QueRenPiLiangShanChuGaoJing")%>' +'</div>'+'<div style="font-size:12px;color:#999999">'+ '<%=rb.getString("PiLiangShanChuGaoJingTiShi")%>' +'</div>';
				vm.$confirm(confirmHint,'<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					dangerouslyUseHTMLString:true
				}).then(function(){
					axios.post('${ctx}/cell/fault/batchClearHistoryAlarm.action',alarmIdsData,{headers:{'Content-Type':'application/json;charset=utf-8'},}).then(function(response){
						var data = response.data;
						if(data["success"]){
							vm.$refs.ctableAlarm.clearSelection();
							vm.$message.success('<%=rb.getString("ChengGong")%>')
							vm.$refs.ctableAlarm.refresh();//表格刷新
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

				if(vm.alarmSelectData.length <= 0)return
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

				if(vm.alarmSelectData.length <= 0)return
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
				var vm = this;

				if(vm.alarmSelectData.length <= 0)return
				var	alarmIds=[];
				vm.alarmSelectData.map((item,index) => {
					alarmIds.push({alarm_id:item.alarm_id,type:item.alarm_type})
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
							vm.$refs.ctableAlarm.clearSelection();
							vm.$message.success('<%=rb.getString("ChengGong")%>')
							vm.$refs.ctableAlarm.refresh();//表格刷新
						}else{
							vm.$message.error('<%=rb.getString("FanQueRenGaoJingTiShi")%><%=rb.getString("TiShiShiBai")%>') //错误提示信息
						}
					})
				}).catch(function(){
					
				})
			},
			/**
			* 模糊查询 -- 活动告警 
			* @param val{string}   查询的val
			*/
			alarmQuery(val){
				var vm = this,
					params ={};
				
				vm.advancedQueryItemList.map((items)=>{
					items.selectVal = '';
					params[items.value] = '';
				})
				Object.assign(vm.params_alarm, params);
				vm.params_alarm.searchText = val;
                vm.params_alarm.uniqueAlarmIdentifier = val;
				vm.search_text = val;
				vm.$refs.ctableAlarm.reset()
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
				vm.sortActive = data.prop ? data.prop : '';
				vm.orderActive = data.order ? order[data.order] : '';
			},
			/**
			* 行点击事件 -- 未读告警标记为一睹
			* @param row{object}   点击行data
			*/
			alarmRowClick(row){
				var vm = this,params={};
				if(row.unread == '1'){
					params.alarmIds = row.alarm_id;
					axios.post('${ctx}/fault/viewConfig/updateUnreadAlarmAlert.action',stringify(params)).then(function(response){
						var data = response.data;
						if(data) {
							if(data["success"]){
							
									vm.$refs.ctableAlarm.refresh();//表格刷新
							
							}else{
								vm.$message.error(data["message"])
							}
						}
					}).catch(function(error){})
				}
			},
			/**
			* 告警列表选中
			* @param selection{Array}   选中数据
			*/
			alarmSelect(selection){
				var vm = this;
				var selectData;
				selectData = selection;
				vm.alarmSelectData = selectData.map((item)=>{
					return Object.assign(item,{alarm_select_name:item.alarm_id+'(' +item.alarm_identifier+ ')'})
				});
			},
			// 左侧列表展开收起事件
			packUpClick(){
				this.packUpClickType = !this.packUpClickType
			},
			// 打开图表详情
			openEchartsDetail(params){
				var vm = this,startTime,endTime,time=params.timeVal,firstDate,lastDate;
				vm.echartLookDetail = true;
				vm.params_lookDetai.alarmType = '';
				vm.params_lookDetai.templateId = '';
				vm.params_lookDetai.searchText = '';
				vm.params_lookDetai_form.searchText = '';
				vm.params_lookDetai.templateId = params.templateId;
				vm.params_lookDetai.alarmType = params.alarmType;
				if(params.timeType == 'day'){

					if(params.tabType == 'add' || params.tabType == 'clear'){
						firstDate = time.split(' ')[0];
						lastDate = time.split(' ')[1];
						startTime = firstDate +' ' + lastDate.split('-')[0];
						endTime = firstDate +' ' + lastDate.split('-')[1];
					}else{
						startTime = '';
						endTime = time;
					}
				}else{
					if(params.tabType == 'add' || params.tabType == 'clear'){
						// endTime = formatDate(addDate(endDate,1));
						// startTime = time +' 00:00:00';
						startTime = time.split(' ~ ')[0];
						endTime = time.split(' ~ ')[1];
					}else{
						var endDate = new Date(time); 
						startTime = '';
						endTime = formatDate(addDate(endDate,1));
					}
				}
				if(params.tabType == 'clear'){
					vm.params_lookDetai.clearStartTime = startTime;
					vm.params_lookDetai.clearEndTime = endTime;
					delete vm.params_lookDetai.eventStartTime
					delete vm.params_lookDetai.eventEndTime
				}else if(params.tabType == 'add'){
					vm.params_lookDetai.eventStartTime = startTime;
					vm.params_lookDetai.eventEndTime = endTime;
					delete vm.params_lookDetai.clearStartTime
					delete vm.params_lookDetai.clearEndTime
				}else if(params.tabType == 'all'){
					vm.params_lookDetai.eventEndTime = endTime;
					delete vm.params_lookDetai.eventStartTime
					delete vm.params_lookDetai.clearStartTime
					delete vm.params_lookDetai.clearEndTime
				}else if(params.tabType == 'active'){
					vm.params_lookDetai.clearStartTime = endTime;
					vm.params_lookDetai.eventEndTime = endTime;
					delete vm.params_lookDetai.eventStartTime
					delete vm.params_lookDetai.clearEndTime
				}
				
			},
			//图表详情模糊搜索
			queryAlarm(){
				var vm = this;
				vm.params_lookDetai.searchText = vm.params_lookDetai_form.searchText;
			},
			// 图表详情导出
			echartsDetailExport(){
				var	vm = this,
					params={},
					urls ='${ctx}/fault/view/exportViewPageList.action';
				params = Object.assign(params,vm.params_lookDetai);
				exportByForm(urls,params);
			},
			// 告警图表加载成功回调
			tableLoadSuccess(){
				var vm = this;
				vm.loading = false;
			},
			//初始化 告警类型值
			initAlarmType(){
				var vm = this,
					alarmTypeList=[{label:'<%=rb.getString("QuanBu")%>',value:''},];
				
				axios.post("${ctx}/cell/fault/queryAlarmType.action").then((res) => {
					var data = res.data.AlarmType ? res.data.AlarmType : [];
					data.map((item)=>{
						if(item.toUpperCase() == 'EGW'){
							alarmTypeList.push({label:'WCG',value:item.toUpperCase()})
						}else{
							alarmTypeList.push({label:item.toUpperCase(),value:item.toUpperCase()})
						}
					})
					vm.advancedQueryItemList.map((items)=>{
						if('neType' == items.value){
							items.options = alarmTypeList
						}
					})
				});
                let newDayTime = formatDate(new Date(gloableTime));
                let startDayTime = formatDate(addDate(newDayTime, -2)).slice(0,10) + ' 00:00:00';
                let endDayTime = newDayTime.slice(0,10) + ' 23:59:59';
                vm.queryTableTime = [startDayTime, endDayTime];
                vm.params_alarm.eventStartTime = startDayTime;
                vm.params_alarm.eventEndTime = endDayTime;

				vm.$nextTick(function(){
					vm.alarmDetailUrl = '${ctx}/fault/view/queryViewPageList.action';
				})
			},
			// 告警库搜索事件
			queryLibrary(val){
				var vm = this;
				vm.libraryParams.search_text = val;
			},
			// 告警库导出
			exportAlarmLib(){
				var vm = this;
				exportByForm("${ctx}/cell/fault/exportAlarmLevelResult.action",{
					searchText: vm.libraryParams.search_text,
					sort:vm.libSort,
					order:vm.libOrder,
					timeZone: timeZone
				})
			},
			// 告警库严重程度级别重置
			resetAlarmServerity(serverityId,alarmIdentifier){
				var vm = this,
					operatorCode = operatorCodeGloab,
					params={
						operator_code: operatorCode,
						serverityId: serverityId,
						alarmIdentifier:alarmIdentifier
					};
				vm.$confirm('<%=rb.getString("QueDingXiuGaiGaoJingJjiBie")%>','<%=rb.getString("QueRen")%>').then(function(){

							axios.post('${ctx}/cell/fault/updateAlarmServerity.action',stringify(params)).then(function(response){
								var data = response.data;
								if(data) {
									if(data["success"]){
										vm.$message({
											message: '<%=rb.getString("ChengGong")%>',
											type:'success'
										});
										refreshAliveAlarmCount();
										vm.$refs.libraryTable.refresh()
										
									}else{
										vm.$message.error(data["message"])
									}
								}
							}).catch(function(error){})
						});
			},
			// 告警库排序
			librarySortChange(data){
				var vm =this,
					order={
						ascending:'asc',
						descending:'desc'
				};
				vm.libSort = data.prop ? data.prop : '';
				vm.libOrder = data.order ? order[data.order] : '';
			},
            queryInputTypeChange(){
                var vm = this;
                if(vm.queryInputType == 'Fuzzy'){
                    vm.params_alarm.searchText = this.search_text;
                    vm.params_alarm.uniqueAlarmIdentifier = '';
                }else{
                    vm.params_alarm.searchText = '';
                    vm.params_alarm.uniqueAlarmIdentifier = this.search_text;
                }
        
            },
			// 模糊搜索
			query(){
				var vm = this;
                if(vm.queryInputType == 'Fuzzy'){
                    vm.params_alarm.searchText = this.search_text;
                }else{
                    vm.params_alarm.uniqueAlarmIdentifier = this.search_text;
                }
			},
			// 搜索域聚焦事件
			queryInputFocus(){
				var vm = this;
                if(vm.queryInputType == 'Fuzzy'){
                    if(northOperatorScenario == 'S0009'){
                        vm.placeholderText = '<%=rb.getString("XuHao")%>/<%=rb.getString("KeNengYuanYin")%>/<%=rb.getString("WangYuanDingWei")%>/<%=rb.getString("ZhanZhiMingCheng")%>';
                    }else{
                        vm.placeholderText = '<%=rb.getString("XuHao")%>/<%=rb.getString("KeNengYuanYin")%>/<%=rb.getString("WangYuanDingWei")%>';
                    }
                }else{
                     vm.placeholderText = '<%=rb.getString("GaoJingWeiYiBiaoZhi")%>';
                }
				
			},
			// 搜索域失焦事件
			queryInputBlur(){
				var vm = this;
				vm.placeholderText = '<%=rb.getString("QingShuRu")%>';
			},
			// 高级查询 确定事件
			advanceQuery(type,paramsItem,value){
				var vm = this,
					params ={};
				if(type == 'select'){
					params[paramsItem] = value;
				}else{
					params[paramsItem] = value.join(',');
				}
				Object.assign(vm.params_alarm, params);
			},
			// 高级搜索下拉全选事件
			handleCheckedAllChange(item){
				var vm = this,
					allList = [];
				item.options.map((items)=>{
					if(items.value){
						allList.push(items.value)
					}
				})
				vm.advancedQueryItemList.map((items)=>{
					if(item.value == items.value){
							items.checkedItemList = items.checkAll? allList : [];
							items.isIndeterminate = false;
					}
				})
			},
			// 高级搜索下拉单选事件
			handleCheckedChange(item){
				var vm = this,
					checkCount = item.checkedItemList.length,
					allCount = item.options.length;
				
				vm.advancedQueryItemList.map((items)=>{
					if(item.value == items.value){
						if(items.type == 'checkbox'){
							items.checkAll = checkCount === allCount;
							items.isIndeterminate = checkCount > 0 && checkCount < allCount;
						}
					}
				})
			},
			// 高级搜索下拉弹窗 展开事件
			checkPopoverShow(item){
				var vm = this;
				vm.advancedQueryItemList.map((items)=>{
					if(item.value == items.value){
						items.popoverShow = true;
					}
				})
			},
			// 高级搜索下拉弹窗 收起事件
			checkPopoverHide(item,types){
				var vm = this,
					checkCount = 0,
					allCount = item.options.length;
				if(item.type != 'select'){
					checkCount = item.oldCheckedItemList.length
				}
				vm.advancedQueryItemList.map((items)=>{
					if(item.value == items.value){
						items.popoverShow = false;
						if(items.type != 'select'){
							items.checkedItemList = items.oldCheckedItemList;
						}
						if(items.type == 'checkbox'){
							items.checkAll = checkCount === allCount;
							items.isIndeterminate = checkCount > 0 && checkCount < allCount;
						}
					}
				})
				if(types == 'del'){
					document.body.click();
				}
			},
			// 高级搜索下拉弹窗 选择提交
			checkPopoverSubmit(item){
				var vm = this,
					params ={};
				if(item.type == 'checkbox'){
					vm.advancedQueryItemList.map((items)=>{
						if(item.value == items.value){
							items.oldCheckedItemList = items.checkedItemList;
							if(items.checkAll){
								params[items.value] = '';
							}else{
								params[items.value] = items.checkedItemList.join(',');
							}
						}
					})
				}else{
					vm.advancedQueryItemList.map((items)=>{
						if(item.value == items.value){
							items.oldCheckedItemList = items.checkedItemList;
						}else{
							var result = item.checkedItemList.includes(items.value);
							if(!result){
								items.isShow = false;
								items.checkedItemList = [];
								items.oldCheckedItemList = [];
								items.checkAll = false;
								items.isIndeterminate = false;
								params[items.value] = '';
							}else{
								items.isShow = true;
							}
						}
					})
				}
				
				Object.assign(vm.params_alarm, params);
				document.body.click();
			},
			// 筛选项删除事件
			checkItemDel(item){
				var vm = this,
					params ={};
				vm.advancedQueryItemList.map((items)=>{
					if(item.value == items.value){
						items.isShow = false;
						if(items.type != 'select'){
							items.checkedItemList = [];
							items.oldCheckedItemList = [];
							items.checkAll = false;
							items.isIndeterminate = false;
						}else{
							items.selectVal = '';
						}
						params[items.value] = '';
					}
					if(items.value == 'add_filter'){
						var idx = items.checkedItemList.indexOf(item.value),
							oldIdx = items.oldCheckedItemList.indexOf(item.value);
						if(idx>-1){
							items.checkedItemList.splice(idx,1)
						}
						if(oldIdx>-1){
							items.oldCheckedItemList.splice(idx,1)
						}
					}
				})
				Object.assign(vm.params_alarm, params);
			},
			// 筛选单选 选择事件
			selectChangeClick(value,item){
				var vm = this,
					params ={};
				
				vm.advancedQueryItemList.map((items)=>{
					if(item.value == items.value){
						items.selectVal = value;
						params[items.value] = value;
					}
				})
				Object.assign(vm.params_alarm, params);
				document.body.click();
			},
            // 时间选择器改变事件
            queryTableTimeChange(val){
                var vm = this;
                vm.queryTableTime = val ? val : [];
                if(val != null){
                    vm.params_alarm.eventStartTime = val[0];
                    vm.params_alarm.eventEndTime = val[1];
                }else{
                    vm.params_alarm.eventStartTime = '';
                    vm.params_alarm.eventEndTime = '';
                }
            },
			// 清除筛选
			clearFilterClick(){
				var vm = this,
					params = {};
				vm.advancedQueryItemList.map((items)=>{
					if(items.isShow && items.isShow== true ){
						if(items.type == 'checkbox'){
							items.checkedItemList = [];
							items.oldCheckedItemList = [];
							items.checkAll = false;
							items.isIndeterminate = false;
						}else if(items.type == 'select'){
							items.selectVal = '';
						}
						if(items.type == 'checkbox' || items.type == 'select'){
							params[items.value] = '';
						}
					}
				})
				Object.assign(vm.params_alarm, params);
				document.body.click();
			},
			resetQueryParams(objs){
				var vm = this,
					params = {
						eventStartTime : '',
						eventEndTime : '',
                        unread: objs.unread,
                        searchText : objs.search_text,
						alarmServerity: objs.alarm_severity,
                        uniqueAlarmIdentifier: objs.uniqueAlarmIdentifier,
					};
				vm.totalChildNum = '2';
				vm.defaultTotalChildNum = '2';
				vm.totalClick('2');
                vm.queryTableTime = [];
                vm.search_text = params.searchText;
                if(params.uniqueAlarmIdentifier){
                    vm.queryInputType = 'Precise';
                    vm.search_text = params.uniqueAlarmIdentifier;
                }
				vm.advancedQueryItemList.map((items)=>{
					if(items.isShow && items.isShow== true ){
						if(items.type == 'select'){
							if(items.value == 'alarmServerity'){
								items.selectVal = params.alarmServerity;
							}else if(items.value == 'unread'){
								items.selectVal = params.unread;
							}else{
								items.selectVal = '';
							}
						}
					}
				});
				Object.assign(vm.params_alarm, params);
				vm.$refs.viewAddSlide.hide();
				vm.$refs.sharingSlide.hide();
				vm.$refs.alarmFilterAddSlide.hide();
			}
		},
		mounted(){
		 	this.initAlarmType();
			eventBus.$off("close-add").$on("close-add",this.cancelAdd);
			eventBus.$off("open-chartDetail").$on("open-chartDetail",this.openEchartsDetail);
			eventBus.$off("open-filter-add").$on("open-filter-add",this.openAddFilterTmp);
			eventBus.$off("close-filter-add").$on("close-filter-add",this.cancelFilterTmpAdd);
		},
})
</script>