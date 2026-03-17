<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" language="java" %>
<%@ include file="/common/taglibs.jsp" %>

<style>
	.container .slide-position-top .el-icon-close {
		font-size: 14px !important;
	}
	#gnbPlugAndPlayPage .resultHeadBox .statisticsSuccessDiv{
		height: 14px;
		line-height: 14px;
		display: flex;
		font-size: 14px;
		border-right: 1px solid #DFE2EE;
	}
	#gnbPlugAndPlayPage .resultHeadBox .statisticsFailDiv{
		height: 14px;
		line-height: 14px;
		display: flex;
		font-size: 14px;
	}
	#gnbPlugAndPlayPage .statisticsSuccessDiv .el-icon::before{
		color:#67D972;
	}
	#gnbPlugAndPlayPage .statisticsSuccessDiv > div:first-child{
		padding: 0px 10px;
		color: #67D972;
	}
	#gnbPlugAndPlayPage .statisticsSuccessDiv > div:last-child{
		padding: 0px 10px 0 0;
	}
	
	#gnbPlugAndPlayPage .statisticsFailDiv .el-icon::before{
		color: #E88282;
		content:'\e6fb';
	}
	#gnbPlugAndPlayPage .statisticsFailDiv > div:first-child{
		padding: 0px 10px;
		color: #E88282;
	}

	i.disabled { opacity: 0.6; }
	#gnbPlugAndPlayPage .commonBorder { border: 1px solid #D5DCEC; border-radius: 10px; }
	#gnbPlugAndPlayPage .commonToolBarBox .el-query .advanceQuery .el-icon { font-size: 14px; }
	#gnbPlugAndPlayPage .commonToolBarBox .el-query .advanceQuery { height: 26px; border-radius: 8px; }
	#gnbPlugAndPlayPage .commonToolBarBox .el-query .advanceQuery .el-input.el-input--small{ width: 280px; }
	#gnbPlugAndPlayPage .commonToolBarBox .el-query .advanceQuery .el-input.el-input--small .el-input__inner { height: 26px; line-height: 26px; }
	#gnbPlugAndPlayPage .commonToolBarBox { height: 30px; }
	#gnbPlugAndPlayPage .commonToolBarBox .el-query .advanceQuery .el-icon:before,
	#gnbPlugAndPlayPage .commonIcon:before,
	#gnbPlugAndPlayPage .informationWarpSlide .el-card__header { color: #7A7992; }
	
	
	#gnbPlugAndPlayPage .eNBTableTitle { 
		line-height: 30px;
		padding-left: 20px;
	}
	
	#gnbPlugAndPlayPage .el-table th { background: #F9F9F9; }
	#gnbPlugAndPlayPage .el-table td { color: #666666; }
	#gnbPlugAndPlayPage .el-table--border th { border-right: 1px solid #E9E9E9; }
	#gnbPlugAndPlayPage .el-table th>.cell { color: #999999; font-weight: normal; }

	#gnbPlugAndPlayPage .radioBox .el-radio-button__inner { padding: 0 12px; border: none; line-height: 24px; font-size: 12px; }
	#gnbPlugAndPlayPage .radioBox .el-radio-button:first-child .el-radio-button__inner { border-radius: 100px 0 0 100px; border-left: none; }
	#gnbPlugAndPlayPage .radioBox .el-radio-button:last-child .el-radio-button__inner { border-radius: 0 100px 100px 0; }
	#gnbPlugAndPlayPage .radioBox .el-radio-button__orig-radio:checked+.el-radio-button__inner {
		color:var(--main-color);
		background: rgba(var(--main-color-rgba1),0.1);
		border-color:var(--main-color) !important;
		border-radius: 100px;
	}
	#gnbPlugAndPlayPage .radioBox { border: 1px solid #E9E9E9; height: 26px; border-radius: 100px; background: #FFFFFF !important;}

	#gnbPlugAndPlayPage .executeStatusQuery .el-query{ right: 0; }
	#gnbPlugAndPlayPage .statusSize { font-size: 18px; margin-right: 4px; margin-top: 2px; }
	#gnbPlugAndPlayPage .el-icon-status-disable:before { color: #B8C3D9; }
	#gnbPlugAndPlayPage .el-icon-status-enable:before { color: #4ED76E; }
	#gnbPlugAndPlayPage .statusText { font-size: 12px; color: #666666; }
	#gnbPlugAndPlayPage .el-progress-bar { width: 85px; }
	#gnbPlugAndPlayPage .el-progress__text { display: none; }
	#gnbPlugAndPlayPage .runningStatus {  width: 90px; background: #E4F1FF;  color: #4D84FF; }
	#gnbPlugAndPlayPage .successStatus {  width: 76px; background: #EEFFF3;  color: #4ED76E;}
	#gnbPlugAndPlayPage .skipStatus { width: 90px; background: #FFFDE3;  color: #FFAA00;}
	#gnbPlugAndPlayPage .failStatus { width: 50px; background: #FEF2F2;  color: #FF6D59; }
	#gnbPlugAndPlayPage .commonStatus { height: 24px; line-height: 24px; border-radius: 100px; text-align: center; }
	#gnbPlugAndPlayPage .commonWidth166 .el-input { width: 150px; }
	#gnbPlugAndPlayPage .informationWarpSlide { width: calc(100% - 20px) !important; height: 218px !important; margin: 0 8px; bottom: 50px !important; left: 2px !important; }
	#gnbPlugAndPlayPage .informationWarpSlide .el-card { border-radius: 0 0 10px 10px; }
	#gnbPlugAndPlayPage .informationWarpSlide .el-card__header span:last-child { right: 4px !important; }
	#gnbPlugAndPlayPage .informationWarpSlide .el-card__header .el-icon-close {  font-size: 14px; }
	.gnbPlugPlayAddSlide { width: 100% !important; height: 100% !important; border-radius: 10px; }
	.gnbPlugPlayAddSlide .el-card__header { height: 40px !important; line-height: 40px !important; }
	#gnbPlugAndPlayPage .queryGroup { height: 24px; border-radius: 8px; }
	#gnbPlugAndPlayPage .queryGroup .el-input { width: 280px; }
	#gnbPlugAndPlayPage .queryGroup .el-input__inner { width: 280px; height: 24px; line-height: 24px; padding: 0; }
	#gnbPlugAndPlayPage .queryGroup .el-icon-common-search { font-size: 14px; }
	#gnbPlugAndPlayPage .queryGroup .el-icon-common-search:before { color: #7A7992; }
	#gnbPlugAndPlayPage .el-date-editor .el-range-input { font-size: 12px; }
	#gnbPlugAndPlayPage .el-date-editor .el-range__close-icon { margin-top: -12px; }
	#gnbPlugAndPlayPage .specialQuery { margin-left: 10px; }
	#gnbPlugAndPlayPage .specialQuery .el-input__inner { width: 200px;}
	#gnbPlugAndPlayPage .addBtnStyle { top: 14px; margin-right: 0; }
	#gnbPlugAndPlayPage .disabledClass { cursor: not-allowed !important; opacity: 0.4; }
	#gnbPlugAndPlayPage .disabledClass:before {color: #c0c4cc;  cursor: not-allowed !important; }
	#gnbPlugAndPlayPage .defaultClass { cursor: pointer; }

	#gnbPlugAndPlayPage .horizontal-line-gnb {
		cursor: row-resize;
		padding: 2px;
	}
	#gnbPlugAndPlayPage .el-tooltip__popper.is-dark { margin: 0 30px 0 80px; }
	#gnbPlugAndPlayPage .el-tabs__item .el-icon::before {
		color:#333;
	}
	#gnbPlugAndPlayPage .statisticsFailDiv .el-icon-circle-close:before {
		content: '\e6fb'
	}
	#gnbPlugAndPlayPage .el-picker-panel {
		margin: 5px -130px;
	}
	.gnbDetectCheckDialog .editButton{
		position: absolute;
		right: 60px;
		top: 0px;
		padding:0 10px;
		height:24px;
		background:#F2F9FF;
		border-radius:2px;
		line-height:24px;
		cursor:pointer;
		margin-left:10px;
		border:1px solid #1DA3FC;
		display:inline-block;
	}
	.gnbDetectCheckDialog .editButton i{
		font-size:14px !important;
	}
	.gnbDetectCheckDialog .editButton span{
		font-size:12px;
	}
	.gnbDetectCheckDialog .el-pairgrid-title {
		/*display: none !important;*/
		line-height: 50px !important;
	}
</style>
<!-- 即插即用 -->
<div id="gnbPlugAndPlayPage" class="container" style='position: relative;height: 100%;'>
	<!-- 操作按钮 -->
	<div class="circleIcon placeholder-bt addBtnStyle hidden CODE_PLUG_AND_PLAY" placeholder='<%=rb.getString("XinZeng")%>'>
		<span class="el-icon-circle-add el-icon" @click="gnbAddTaskBtnClick"></span>
	</div>
	<div style='height: 100%; display: flex; flex-direction: column;'>
		<!--gnb 策略列表  :data="tbData"-->
		<el-ctable ref="gnbPolicyList" class='commonBorder flex-row-item-gnb' style='margin-bottom: 10px;'
			:url="gnbConfigURL" :pagination="false"
			:query-params="gnbPolicyQuery"
			row-key="policyId"
			@row-click="gnbRowClick"
			@load-success="gnbPolicyListSuccess">
			<div slot="toolbar">
				<div class='commonFlex commonToolBarBox'>
					<div class='commonTextWeight14 eNBTableTitle'><%=rb.getString("CeLueLieBiao")%>({{gnbPolicyNumber}})</div>
					<el-query type="normal" @query="queryPolicyGnb" placeholder='<%=rb.getString("CeLueMingChen") %> / <%=rb.getString("MuBiaoBanBen")%>'></el-query>
				</div>
			</div>
			<el-table-column width="40">
				<template slot-scope="scope">
					<div class="el-icon el-icon-main-more" @click="gnbOptClick(scope.row, event)" v-clickoutside="gnbHanderClose"></div>
				</template>
			</el-table-column>
			<el-table-column label='<%=rb.getString("ShiFouQiYong")%>' prop="switch" width="100" style="position:relative;"><!--switch 换 selfStartEnable-->
				<template slot-scope="scope">
					<el-switch v-model="scope.row.switch" style="height: 18px;"
						:before-change="gnbSwitchChange"
						:active-value="'1'"
						:inactive-value="'0'"
						active-color="#4D84FF"
						inactive-color="#CFCFCF">
					</el-switch>
				</template>
			</el-table-column>
			<el-table-column label='<%=rb.getString("ChanPinLeiXingBiaoZhi")%>' prop="productName" show-overflow-tooltip="true" width="100"></el-table-column>
			<el-table-column label='<%=rb.getString("CeLueMingChen") %>' prop="policyName" show-overflow-tooltip="true"></el-table-column>
			<el-table-column label='<%=rb.getString("ZhiXingFangShi")%>' prop="executeType" width="200">
				<template slot-scope="scope">
					<div v-if="scope.row.executeType === '1'"><%=rb.getString("eNBShouDongZhiXing") %></div>
					<div v-if="scope.row.executeType === '0'"><%=rb.getString("eNBZiDongZhiXing") %></div>
				</template>
			</el-table-column>
			<el-table-column label='<%=rb.getString("RuanJianShengJi")%>' prop="upgradeEnable" show-overflow-tooltip="true"> 
				<template slot-scope="scope">
					<div v-if='scope.row.upgradeEnable == "0"' class='commonFlex'>
						<span class="el-icon el-icon-status-disable statusSize"></span>
						<span class='statusText'><%=rb.getString("JinYong")%> <span v-show='scope.row.destVersion !="" && scope.row.destVersion !=null && scope.row.destVersion !=undefined'><%=rb.getString("MuBiaoBanBen")%>={{scope.row.destVersion}}</span></span>
					</div>
					<div v-if='scope.row.upgradeEnable == "1"' class='commonFlex'>
						<span class="el-icon el-icon-status-enable statusSize"></span>
						<span class='statusText'><%=rb.getString("QiYong")%> <span v-show='scope.row.destVersion !="" && scope.row.destVersion !=null && scope.row.destVersion !=undefined'><%=rb.getString("MuBiaoBanBen")%>={{scope.row.destVersion}}</span></span>
					</div>
				</template>
			</el-table-column>
			<el-table-column label='<%=rb.getString("License")%>' prop="licenseEnable" width="200">
				<template slot-scope="scope">
					<div v-if='scope.row.licenseEnable === "0"' class='commonFlex'>
						<span class="el-icon el-icon-status-disable statusSize"></span>
						<span class='statusText'><%=rb.getString("JinYong")%></span>
					</div>
					<div v-if='scope.row.licenseEnable === "1"' class='commonFlex'>
						<span class="el-icon el-icon-status-enable statusSize"></span>
						<span class='statusText'><%=rb.getString("QiYong")%></span>
					</div>
				</template>
			</el-table-column>
			<el-table-column label='<%=rb.getString("ENBCanShuZiPeiZhi")%>' prop="configEnable" width="200"> <!--selfConfigEnable 变 configEnable 注意这里的逻辑-->
				<template slot-scope="scope">
					<div v-if='scope.row.configEnable == "0"' class='commonFlex'>
						<span class="el-icon el-icon-status-disable statusSize"></span>
						<span class='statusText'><%=rb.getString("JinYong")%></span>
					</div>
					<div v-if='scope.row.configEnable == "1"' class='commonFlex'>
						<span class="el-icon el-icon-status-enable statusSize"></span>
						<span class='statusText'><%=rb.getString("QiYong")%></span>
					</div>
				</template>
			</el-table-column>
		</el-ctable>
		<div class="horizontal-line-gnb"> </div>
		<!-- 执行状态列表-->
		<el-ctable id="gnb_config_status_list" height="76%" style="overflow: auto; flex: auto;" ref="gnbStatusList" class='commonBorder flex-row-item-gnb'
			:time="6"
			row-key="task_id"
			:url="gnbStatusURL"
			:query-params="gnbStatusQuery" @selection-change='gnbBatchSelect'
			@load-success="gnbConfigStatusSuccess">
			<div slot="toolbar">
				<div class='commonFlex' style='margin-bottom: 8px;padding-right: 10px;border-bottom: 1px solid #D5DCEC; position: relative;height: 30px;'>
					<div class='commonTextWeight14 eNBTableTitle' style="line-height: 22px;"><%=rb.getString("ZhiXingZhuangTai")%></div>

					<div class="toolbarHeadBtnBoxCls" style="border-bottom: none; margin: -6px 0 0 0;" v-if="gnbPnpEnable && gnbTabsActive == '0'"> 
						<div class="selectBlukBoxCls">
							<div class="selectMain headBtnItemCls">
								<div class="bulkSelectBtnBoxCls" @click="openBulkSelectTable" style="padding: 0px 0 0 16px; border-right: none;">
									<span class="el-icon-selected el-icon"></span>
									<span class="bulkSelectNumBoxCls">( {{gnbDeviceSelection.length}} )</span>
								</div>
								<div class="selectTableBoxCls" style="position: absolute;top: 26px;left: 0px; height: 258px;" v-show="bulkSelectShow">
									<div class="selectBoxTitle">
										<span><%=rb.getString("YiXuan")%></span>
										<span style="position:absolute;right:20px;top:15px;" class="el-icon el-icon-close" @click="closeBulkSelectTable"></span>
									</div>
									<div class="selectBoxMain">
										<div class="tableInfoCls">
											<div class="tableInfoHeader">
												<div><%=rb.getString("Title_SheBeiBianMa")%></div>
												<div @click="clearBulkSelected"><span style="margin-right:5px;" class="el-icon el-icon-operation-delete" ></span>Clear</div>
											</div>
											<el-ctable 
												id="bulkSelectTable" 
												ref="bulkSelectTable" 
												:data="gnbDeviceSelection" 
												:showHeader="false"
												:rownumber="false"
												:front-pagination="true"
												height="170px" pagination="true" >
												<el-table-column prop="task_id" v-if="false"></el-table-column>
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
						<div :class="gnbDeviceSelection.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" style="border-right: none;" @click="gnbBatchRetryBtn">
							<span class="el-icon el-icon-operation-restart"></span>
							<span><%=rb.getString("ChongXinZhiXing")%></span>
						</div>
					</div>

					<div class="resultHeadBox commonFlex" style='position: absolute; right: 16px; top: 5px;'>
						<div class="statisticsSuccessDiv">
							<div><span class="el-icon el-icon-circle-success" style='margin-right: 6px;'></span><%=rb.getString("ChengGong")%></div>
							<div>{{gnbSuccessCount}}</div>
						</div>
						<div class="statisticsFailDiv">
							<div><span class="el-icon el-icon-circle-close" style='margin-right: 6px;'></span><%=rb.getString("ShiBai")%></div>
							<div>{{gnbFailCount}}</div>
						</div>
					</div>
				</div>
				<div class='commonFlex commonToolBarBox executeStatusQuery' style='position: relative;padding-right: 10px;padding-left: 20px;'>
					<el-radio-group v-model='gnbTabsActive' class='radioBox' @change="gnbTabsChange">
						<el-radio-button label="0"><%=rb.getString("SuoYouRenWu") %></el-radio-button>
						<el-radio-button label="1"><%=rb.getString("RuanJianShengJi") %></el-radio-button>
						<el-radio-button label="2"><%=rb.getString("License") %></el-radio-button>
						<el-radio-button label="3"><%=rb.getString("ENBCanShuZiPeiZhi") %></el-radio-button>
					</el-radio-group>
					<div class='commonFlex' style='margin-left: 10px;'>
						<div class='queryGroup specialQuery' style='width: 220px;'>
							<el-input v-model='statusSearchText' @keyup.enter.native="queryGnbStatus" class='pairgrid-query' placeholder='<%=rb.getString("CeLueMingChen") %> / <%=rb.getString("Title_SheBeiBianMa")%>' style='width: 200px;'></el-input>
							<i @click='queryGnbStatus' class="el-icon el-icon-common-search"></i>
						</div>
						<el-date-picker style='width: 280px; margin: 0 10px;'
							v-model="gnbTimeRange"
							type="datetimerange"
							value-format="yyyy-MM-dd HH:mm:ss"
							range-separator="一"
							@change="gnbDateChange"
							start-placeholder='<%=rb.getString("KaiShiShiJian")%>'
							end-placeholder='<%=rb.getString("JieShuShiJian")%>'>
						</el-date-picker>
						<el-select placeholder='<%=rb.getString("ChanPinLeiXingBiaoZhi") %>' v-model='gnbStatusQuery.productType' class='commonWidth166'>
							<el-option v-for="item in productTypeList" :label="item.name" :value="item.value"></el-option>
						</el-select>
						<el-select placeholder='<%=rb.getString("ZhuangTai") %>' v-model='gnbStatusQuery.status' class='commonWidth166' style='margin: 0 10px;'>
							<el-option label='<%=rb.getString("SuoYou")%>' value=" "></el-option>
							<el-option label='<%=rb.getString("ChengGong")%>' value="0"></el-option>
							<el-option label='<%=rb.getString("ShiBai")%>' value="1"></el-option>
							<el-option label='<%=rb.getString("JinXingZhong")%>' value="2"></el-option>
							<el-option label='<%=rb.getString("WeiZhiXing")%>' value="3"></el-option>
							<el-option label='<%=rb.getString("TiaoGuo")%>' value="4"></el-option>
						</el-select>
					</div>
				</div>
			</div>
			<el-table-column type="selection" :reserve-selection="true" :selectable = "gnbIsDisabled" width="50" v-if="gnbPnpEnable && gnbTabsActive == '0'" :key='9'></el-table-column>
			<!--status: 0-成功， 1-失败，2-进行中，3-未执行（等待）， 4-跳过，  5-部分成功 已去掉-->
			<!--  手动（1）或未执行（3）的任务可以 start;   删除： 等待，失败，成功，部分成功；      重新执行 retry：失败，成功  ;    跳过skip： 可retry ,delete; all: 显示结果页面-->
			<el-table-column width="125" v-if="gnbTabsActive == '0'" :key='6'>
				<template slot-scope="scope">
					<div class='commonFlex'>
						<div>
							<i v-if="(scope.row.status=='0' || scope.row.status=='1' || scope.row.status=='4') && gnbPnpEnable && gnbPolicySwitch == '1'" @click="gnbRestartStatus(scope.row)" :class="[retryLoading==scope.row.policyId  ? 'disabledClass' : 'defaultClass']" class="el-icon el-icon-operation-restart commonIcon defaultClass" title="<%=rb.getString("ChongXinZhiXing")%>"></i>
							<i v-else class="el-icon el-icon-operation-restart commonIcon disabledClass" title='<%=rb.getString("ChongXinZhiXing")%>'></i>
						</div>
						<div style="margin-left: 10px;" v-show="scope.row.execute_type === '1'">
							<!-- 只有手动模式显示 -->
							<i v-if="scope.row.status=='3' && gnbPnpEnable" @click="gnbStartOperStatus(scope.row)" class="el-icon el-icon-operation-start commonIcon defaultClass" title="<%=rb.getString("ZhiXingNew")%>"></i>
							<i v-else class="el-icon el-icon-operation-start commonIcon disabledClass" title='<%=rb.getString("ZhiXingNew")%>'></i>
						</div>
						<div style="margin-left: 10px;">
							<i v-if="(scope.row.status=='0' || scope.row.status=='1' || scope.row.status=='3' || scope.row.status=='4') && gnbPnpEnable" @click="gnbDeletStatus(scope.row)" class="el-icon el-icon-operation-delete commonIcon defaultClass" title="<%=rb.getString("ShanChu")%>"></i>
							<i v-else class="el-icon el-icon-operation-delete commonIcon disabledClass" title='<%=rb.getString("ShanChu")%>'></i>
						</div>
						<div style="margin-left: 10px;">
							<i @click="gnbViewConfigTask(scope.row)" class="el-icon el-icon-operation-info commonIcon" title='<%=rb.getString("XinXi")%>'></i>
						</div>
					</div>
				</template>
			</el-table-column>
			<el-table-column label='<%=rb.getString("Title_SheBeiBianMa")%>' prop="serial_number" width="240" show-overflow-tooltip="true"></el-table-column>
			<el-table-column label='<%=rb.getString("ChanPinLeiXingBiaoZhi")%>' prop="product_type" width="160"></el-table-column>
			<el-table-column label='<%=rb.getString("CeLueMingChen") %>' prop="policy_name" width="200" show-overflow-tooltip="true"></el-table-column>
			<el-table-column label='<%=rb.getString("ZhiXingFangShi")%>' prop="execute_type" width="130">
				<template slot-scope="scope">
					<div v-if="scope.row.execute_type === '1'"><%=rb.getString("eNBShouDongZhiXing") %></div>
					<div v-if="scope.row.execute_type === '0'"><%=rb.getString("eNBZiDongZhiXing") %></div>
				</template>
			</el-table-column>
			<el-table-column label='<%=rb.getString("KaiShiShiJian") %>' prop="start_time" width="150"></el-table-column>
			<el-table-column label='<%=rb.getString("JieShuShiJian") %>' prop="end_time" width="150"></el-table-column>
			<el-table-column label='<%=rb.getString("BuJinDu") %>' prop="execute_procedure" width="210" v-if="gnbTabsActive == '0'" key='execute_procedure'></el-table-column>
			<el-table-column label='<%=rb.getString("ChuShiBanBen") %>' prop="original_version" width="150" key='original_version' v-if="gnbTabsActive == '1'"></el-table-column>
			<el-table-column label='<%=rb.getString("MuBiaoBanBen") %>' prop="target_version" width="150" key='target_version' v-if="gnbTabsActive == '1'"></el-table-column>
			<el-table-column label='<%=rb.getString("LicenseWenJian") %>' prop="lic_file" width="200" key='lic_file' v-if="gnbTabsActive == '2'"></el-table-column>
			<el-table-column label='<%=rb.getString("ZhuangTai")%>' prop="status" width="110">
				<template slot-scope="scope" class="status-info">
					<div class='successStatus commonStatus' v-if="scope.row.status=='0'"><%=rb.getString("ChengGong") %></div>
					<div class='failStatus commonStatus' v-if="scope.row.status=='1'"><%=rb.getString("ShiBai")%></div>
					<div class='runningStatus commonStatus' v-if="scope.row.status=='2'"><%=rb.getString("JinXingZhong")%></div>
					<div class='runningStatus commonStatus' v-if="scope.row.status=='3'"><%=rb.getString("WeiZhiXing") %></div>
					<div class='skipStatus commonStatus' v-if="scope.row.status=='4'"><%=rb.getString("TiaoGuo")%></div>
				</template>
			</el-table-column>
			<el-table-column label='<%=rb.getString("ShiBaiYuanYin")%>' prop="failure_reason" show-overflow-tooltip="true" min-width="200"></el-table-column>
		</el-ctable>
		<el-cmenu ref="gnbMenu" :data="gnbMenus" @click="gnbMenuClick"></el-cmenu>
	</div>
	<!-- Add Config Slider -->
	<el-slide ref="gnbSlider" title='<%=rb.getString("XinZeng")%>'
		:url="slideURL"
		:title="slideTile"
		:header='slideHeader'
		:footer="footerShow" class='gnbPlugPlayAddSlide'
		@ok="gnbAddConfig"
		@cancel="gnbCloseConfig">
	</el-slide>

	<!-- gnb execute status 详情 -->
	<el-slide ref="slideView" title="Devices Execute Information" :footer="false" position="bottom" class='informationWarpSlide'
	    height="300px" :modal='modal' @cancel='gnbCancelViewSlide'>
	    	<el-ctable id="exeProgressTable" ref="ctableProgress" :url="taskUrl" :query-params="params_progress" time="6"
				:row-key="'id'" :pagination="false" :rownumber="false">
				<el-table-column label='<%=rb.getString("JinDu")%>' prop="progress">
					<template slot-scope="scope" class="status-info">
						<div v-if="scope.row.progress=='1'"><%=rb.getString("RuanJianShengJi")%></div>
						<div v-else-if="scope.row.progress=='2'"><%=rb.getString("License")%></div>
						<div v-else-if="scope.row.progress=='3'"><%=rb.getString("ENBCanShuZiPeiZhi") %></div>
						<div v-else-if="scope.row.progress=='4'">Cell active</div>
					</template>
				</el-table-column>
				<el-table-column label='<%=rb.getString("ZhuangTai")%>' prop="status">
					<template slot-scope="scope" class="status-info">
						<div class='successStatus commonStatus' v-if="scope.row.status=='0'"><%=rb.getString("ChengGong") %></div>
						<div class='failStatus commonStatus' v-else-if="scope.row.status=='1'"><%=rb.getString("ShiBai")%></div>
						<div class='runningStatus commonStatus' v-else-if="scope.row.status=='2'"><%=rb.getString("JinXingZhong")%></div>
						<div class='runningStatus commonStatus' v-else-if="scope.row.status=='3'"><%=rb.getString("WeiZhiXing") %></div>
						<div class='skipStatus commonStatus' v-else-if="scope.row.status=='4'"><%=rb.getString("TiaoGuo")%></div>
					</template>
				</el-table-column>
				<el-table-column label='<%=rb.getString("KaiShiShiJian")%>' prop="start_time"></el-table-column>
				<el-table-column label='<%=rb.getString("JieShuShiJian")%>' prop="end_time"></el-table-column>
				<el-table-column label='<%=rb.getString("ShiBaiYuanYin")%>' prop="failure_reason" show-overflow-tooltip="true"></el-table-column>
			</el-ctable>
	 </el-slide>

	 <el-dialog ref="gnbCheckList" :visible.sync="gnbCkDLShow" width='1100px' :append-to-body="true" :close-on-click-modal="false" @close='gnbCkDLShow = false' class="gnbDetectCheckDialog">
		<div slot="title" style="padding: 15px 0;display: flex;align-items: center;">
			<span class="commonText14"><%=rb.getString("GNBSheBeiLieBiao") %></span>
			<span class="commonNotes12"> (<%=rb.getString("SheBeiJianCheTiShi") %>)</span>
		</div>
		<el-form ref='gnbSelectSnListForm' :rules='gnbSelectSnListRules' :model='gnbSelectSnListForm' label-position="top">
			<el-pairgrid :id="'gnbDetectTb'"
				:rownumber="true"
				ref="gnbDetectTb"
				:left-url="gnbDetectURL"
				:height="'400px'"
				row-key="serialNumber"
				:query-params="gnbDetectParams"
				@selection-change='gnbSelectChange'>
				
				<template slot="left">
					<el-table-column type="selection" :reserver-selection="true"></el-table-column>
					<el-table-column width="45" prop="connection_status">
						<template slot-scope="scope">
							<div v-html="connStatusFormatter(scope.row.connection_status,scope.row)"></div>
						</template>
					</el-table-column>
					<el-table-column label='<%=rb.getString("Title_SheBeiBianMa") %>' prop="serialNumber" min-width="120"></el-table-column>
					<el-table-column label='<%=rb.getString("GNBMingCheng") %>' prop="cellName" min-width="100"></el-table-column>
					<el-table-column label='<%=rb.getString("BanBenHao") %>' prop="softwareVersion" min-width="100"></el-table-column>
					<el-table-column label='<%=rb.getString("ChanPinLeiXingBiaoZhi") %>' prop="product" min-width="80"></el-table-column>
					<el-table-column label='<%=rb.getString("SheBeiZu") %>' prop="groupName" min-width="120"></el-table-column>
				</template>
				<template slot="toolbar">
					<div style="position: relative;">
						<div class="searchCon" style="margin-left:18px;">
							<el-input placeholder='<%=rb.getString("Title_SheBeiBianMa")%>' suffic-icon='el-icon-search' v-model="gnbDetectSearchText" style="width:270px;">
								<i slot="suffix" class="el-icon el-icon-common-search searchCls" @click="gnbQueryDetect"></i>
							</el-input>
						</div>
						<div class="editButton" @click="gnbAddBatchSnClick" size="mini">
							<i class="el-icon el-icon-batchInput" style="font-size: 14px;padding-right: 5px;"></i>
							<span><%=rb.getString("PiLiangShuRu")%></span>
						</div>
					</div>
				</template>
				<template slot="right">
					<el-table-column label='<%=rb.getString("Title_SheBeiBianMa") %>' prop="serialNumber" min-width="120"></el-table-column>
					<el-table-column label='<%=rb.getString("GNBMingCheng") %>' prop="cellName" min-width="100"></el-table-column>
				</template>
			</el-pairgrid>
			<el-form-item prop='serialNumber' style="margin: 0;" label-width="0">
				<el-input v-model='gnbSelectSnListForm.serialNumber' v-show="false"></el-input>
			</el-form-item>
		</el-form>
		
		<div style="padding-top: 30px;">
			<el-button type="primary" @click="gnbSendDetection"><%=rb.getString("QueDing") %></el-button>
			<el-button @click="gnbCkDLShow = false"><%=rb.getString("QuXiao") %></el-button>
		</div>
	</el-dialog>
	<!--batch input-->
	<el-dialog class='dialogStyle' title='<%=rb.getString("TianJia")%>' width='630px' :visible.sync='gnbBatchSnDialog' :append-to-body="true" :close-on-click-modal="false" @close='gnbCloseBatchSn'>
		<el-form ref='gnbBatchSnForm' :rules='gnbBatchSnRules' :model='gnbBatchSnForm' label-position="top">
			<div>
				<label><%=rb.getString("Title_SheBeiBianMa")%></label>
				<el-form-item prop='serialNumber' style="margin-bottom:22px;">
					<el-input v-model='gnbBatchSnForm.serialNumber' type='textarea' :rows="4" style='margin-top:5px;'></el-input>
				</el-form-item>
				<p style='display:flex;color:#BBB'><span class='el-icon el-icon-circle-info' style='font-size:14px;'></span><span style="font-size:12px;"><%=rb.getString("eNBZhuCeTiShiWenZi") %></span></p>
			</div>
			<div style='margin-top:45px;'>
				<el-button @click='gnbSaveBatchSn' type="primary"><%=rb.getString("QueDing")%></el-button>
				<el-button @click='gnbCloseBatchSn'><%=rb.getString("QuXiao")%></el-button>
			</div>
		</el-form>
	</el-dialog>

	<!-- retry-->
	<el-dialog title='<%=rb.getString("QueRen")%>' width="600px" :visible="gnbShoBatchRetryShow" :close-on-click-modal="false" :modal-append-to-body="false" @close="closeGnbBatchRetry">
		<el-form label-position="top" ref="gnbBatchRetryForm" :model='gnbBatchRetryForm'>
         	<div class="commonSize14"><%=rb.getString("ChongShiRenWu")%></div>
         	<div class="commonNotes12" style="padding: 10px 0;"><%=rb.getString("WuFaChongShiTiShi")%></div>
			<el-checkbox v-model="gnbBatchRetryForm.isContainSuccessTask" :true-label="1" :false-label="0" style="line-height: 14px;"><%=rb.getString("TongShiZhiXingChengGongRenWu")%></el-checkbox>
        </el-form>
       	<div slot="footer">
          	 <el-button type="primary" @click="gnbBatchRetryConfirm"><%=rb.getString("QueDing")%></el-button>
             <el-button @click="closeGnbBatchRetry"><%=rb.getString("QuXiao")%></el-button>
        </div>			
	</el-dialog>
</div>

<script>

	var gnbPlugAndPlayVue = new Vue({
		el: '#gnbPlugAndPlayPage',
		data() {
			var gnbValidatorSn = (rule,value,callback) => {
				var serialNumber = value, temp = /^(\d|[a-zA-Z]|-|\s){1,30}$/,
					list = serialNumber.replace(/[(\r\n)\r\n\s；]+/g,';').split(';').filter(function(item){ 
						return item.length > 0;
					});
				
				if (serialNumber == null || serialNumber.length == 0) {
					callback(new Error('<%=rb.getString("SNBuNengWeiKong")%>'));
				}else {
					var nameFlag = list.every(function(item,index){
						return temp.test(item)
					})
					if(nameFlag){
						callback()
					}else{
						callback(new Error('<%=rb.getString("QingShuRuZhengQueSn")%>'));
					}
				}
			},
			validatorSNs = (rule,value,callback) => {
				var vm = this;
				
				if(vm.gnbDetectSelections.length == 0) {
					callback('<%=rb.getString("QingXuanZeJiLu")%>');
				}else {
					callback();
				}
			};
			return {
				//策略列表数据
				tbData: [
					{
						'policyId':'e8ee001',
						'policyName':'ssw_20230922',
						'switch':'0',
						'productType':'BaiBNQ',
						'executeType':'1',
						'upgradeEnable':'1',
						'licenseEnable':'1',
						'configEnable':'1',
						'destVersion':'BaiBu_BNQ'
					}
				],
				gnbConfigURL: '${ctx}/gnb/pnp/queryGnbPnPPolicyPageList.action',
				gnbPolicyQuery: {
					searchText: '',
					timeZone: timeZone, //注意是否需要此参数
				},
				gnbPolicyNumber: 0,
				gnbMenus: [],
				gnbStatusURL: '${ctx}/gnb/pnp/getPnPTaskPageList.action',
				gnbStatusQuery: {
					searchText: '',
					productType: '',
					status: '',
					queryStartTime: '',
					queryEndTime: '',
					timeZone: timeZone
				},
				statusSearchText: '',
				gnbSuccessCount: 0,
				gnbFailCount: 0,
				gnbTabsActive: '0',
				productTypeList: [],
				gnbTimeRange: [],

				productModels: [],
				slideHeader: false,
				footerShow: false,
				gnbCurrentRow: [],
				slideTile: '',
                slideURL: '',

                slideUrl:"",
				slideTitle:"",
				slideFooter:"",
				slideHeader:"",
				slidePosition:"",
				slideHeight:"",
				slideWidth:"",
				slideModal:"",
				taskUrl:"",
				params_progress:{
					timeZone:timeZone,
					task_id: '',
					page: 1,
					rows: 50
				},

				gnbDetectURL: '',
				gnbSelectSnListForm:{
					serialNumber:''
				},
				gnbSelectSnListRules:{
					serialNumber:[
						//{validator:validatorSNs,trigger:'change'}
						{required:true,message:'<%=rb.getString("QingXuanZeJiLu")%>',trigger:'change'}
					]
				},
				gnbDetectParams: {
					policyId: '',
					searchText: ''
				},
				gnbCkDLShow: false,
				gnbDetectSelections: [],
				retryLoading: '',
				gnbDetectSearchText: '',
				gnbPolicySwitch: '1',
				bulkSelectShow: false,
				gnbDeviceSelection: [],
			 	//retry 确认
				gnbShoBatchRetryShow:false,
				gnbBatchRetryForm: {
					isContainSuccessTask: '0'
				},
				//batch input sn
				gnbBatchSnDialog: false,
				gnbBatchSnForm:{
					serialNumber: '',
					type: 'input'
				},
				gnbBatchSnRules:{
					serialNumber:[
						{validator: gnbValidatorSn,trigger:'change'}
					]
				},
			}
		},
		watch: {
			gnbTimeRange(newVal){
				var vm = this;
				if(!newVal){
					newVal = [];
					vm.gnbStatusQuery.queryStartTime = '';
	    			vm.gnbStatusQuery.queryEndTime = '';
				}
			},
			gnbDetectSelections(){
				var vm = this,
					data = vm.$refs.gnbDetectTb.getData();
			
				if(data && data.length > 0){
					vm.gnbSelectSnListForm.serialNumber = data.map(function(row){ return row.serialNumber ;}).join(',');
				}else{
					vm.gnbSelectSnListForm.serialNumber = '';
				}
			}
		},
		computed: {

			gnbPnpEnable() {

				return writableMap['CODE_PLUG_AND_PLAY'] == true;
			}
		},
		methods: {
			gnbInit(){
				var vm = this;

				//gNB获取产品类型
				axios.post("${ctx}/gnb/pnp/getProductTypeSelect.action").then(function(res){
					var data = res.data;
					
			   		var arr = [{name:'<%=rb.getString("QuanBu")%>',value:''}];
			   		(data || []).map(function(item){
			   			if (item){
			   				arr.push({name:item.name,value:item.value})
			   			}
			   		})
			   		vm.productTypeList = arr;
				});
			},

			//GNB Add Policy Task
			gnbAddTaskBtnClick() {
				var vm = this,
					rd = Math.random(),
					url = '${ctx}/gnb/pnp/toGNBAddConfig.action?r='+rd;
					
				vm.slideTile = '<%=rb.getString("XinZeng")%> Policy';
				vm.slideURL = url;
				vm.slideHeader = false;
				vm.footerShow = false;
				vm.$refs.gnbSlider.showSlide(function(){
					eventBus.$emit('gnbInit-plugPlayConfig','add','');
				});
				vm.gnbCancelViewSlide();
			},
			//gnb Add or modify Policy Save
			gnbAddConfig() {
				eventBus.$emit('gnbSave-plugPlayConfig');
			},
			//gnb Add or modify Policy close
			gnbCloseConfig() {
				var vm = this;
				eventBus.$emit('gnbHandle-plugPlayCancel')
			},
			//refresh  gnb table
			gnbReloadConfig() {
				var vm = this;

				vm.$refs.gnbPolicyList.refresh();
			},
			//gnb gnbRowClick
			gnbRowClick(row) {
				this.gnbCurrentRow = row;
			},
			//gnb Policy list search
			queryPolicyGnb(text) {
				this.gnbPolicyQuery.searchText = text;
			},
			//gnb Execute Status search
			queryGnbStatus(text) {
				this.gnbStatusQuery.searchText = this.statusSearchText;
			},
			//gnb time search
	    	gnbDateChange(val) {
				var vm = this;
				vm.gnbTimeRange = val;
				if(val != null){
    				this.gnbStatusQuery.queryStartTime = vm.gnbTimeRange[0];
	    			this.gnbStatusQuery.queryEndTime = vm.gnbTimeRange[1];
    			}
			},
			//gnb policy list loadSuccess
			gnbPolicyListSuccess(data) {
				var vm = this,
				properties = data.total || 0;
				vm.gnbPolicySwitch = data.rows[0].switch;
				vm.gnbPolicyNumber = properties || 0;
			},

			//gnb Execute Status loadSuccess
			gnbConfigStatusSuccess(data) {
				var vm = this,
					properties = data.properties||{};

				vm.gnbSuccessCount = properties.successCount||0;
				vm.gnbFailCount = properties.failCount||0;
			},
			//gnb Execute Status tabs
			gnbTabsChange(val){
				var vm = this;
				vm.gnbStatusQuery =  {
					searchText: '',
					productType: '',
					status: '',
					queryStartTime: '',
					queryEndTime: '',
					timeZone: timeZone
				};
				vm.gnbTimeRange = [];
				vm.statusSearchText = '';

				if(val == '0'){
					vm.gnbStatusURL= '${ctx}/gnb/pnp/getPnPTaskPageList.action';
				}else {
					vm.gnbStatusURL= '${ctx}/gnb/pnp/getPnPTaskRecordPageList.action';
				}

				if(val == '1'){
					//升级
					vm.gnbStatusQuery.progress = '1';
				}else if(val == '2'){
					//license
					vm.gnbStatusQuery.progress = '2';
				}else if(val == '3'){
					//参数配置
					vm.gnbStatusQuery.progress = '3';
				}
				vm.gnbCancelViewSlide();
			},
			//gnb enable switch click
			gnbSwitchChange(fn) {
				var vm = this, params = {},
					url =  '${ctx}/gnb/pnp/updatePnPPolicySwitch.action';

				vm.$nextTick(function(){
					if(vm.gnbCurrentRow.switch == '0'){
						params.policySwitch= '1'; //注意这里 selfStartEnable 换 policySwitch
					}else{
						params.policySwitch= '0';
					}

					params.policyId= vm.gnbCurrentRow.policyId;
					params.policyName = vm.gnbCurrentRow.policyName;

					axios.post(url, stringify(params)).then(function(res){
						var data = res.data;

						if(data.success) {
							fn();
							vm.$message({
								message: '<%=rb.getString("ChengGong")%>',
								type:'success'
							});

							vm.$refs.gnbPolicyList.refresh();
						}else {
							vm.$message({
								message: data['message'],
								type: 'error'
							});
						}

					}).catch(function(){})
				})
			},
			//gnb Execute Status: view info
			gnbViewConfigTask(row){
				var vm = this;
				vm.params_progress.task_id = row.task_id;
				vm.taskUrl = "${ctx}/gnb/pnp/getPnPTaskRecordPageList.action";
				vm.$refs.ctableProgress.refresh();
				vm.$refs.slideView.showSlide(function(){
	    	    	vm.slideModal = false;
	    	    });
			},
			//gnb Execute Status: retry
			gnbRestartStatus(row){
				var vm = this, params = {};

					params = {
						taskId: row.task_id
					};

				vm.retryLoading = row.policyId;
				axios.post('${ctx}/gnb/pnp/updatePnPManualReExecutePolicyTaskStart.action', stringify(params)).then(function(res){
					var data = res.data;

					if(data['success']) {
						vm.$message({
							message: '<%=rb.getString("ChengGong")%>',
							type: 'success'
						});

						vm.$refs.gnbStatusList.refresh();
					}else {
						vm.$message({
							message: data['message'],
							type: 'error'
						});
					}
					vm.retryLoading = '';
				}).catch(function(){});
				vm.gnbCancelViewSlide();
			},
			//retry按钮点击
			gnbBatchRetryBtn(){
				var vm = this;
				if(vm.gnbDeviceSelection.length == 0){
					return
				}else{
					vm.gnbBatchRetryForm.isContainSuccessTask = '0';
					vm.gnbShoBatchRetryShow = true;
				}
			},
			//确认 retry
			gnbBatchRetryConfirm(){
				var vm = this, params = {}, taskIds=[];
				
				if(vm.gnbDeviceSelection.length == 0){
					return
				}else{
					taskIds = vm.gnbDeviceSelection.map(function(item){ return item.task_id});
					
					params = {
						taskId: taskIds.join(','),
						isContainSuccessTask: vm.gnbBatchRetryForm.isContainSuccessTask
					};
				}
				
				axios.post('${ctx}/gnb/pnp/updatePnPManualReExecutePolicyTaskStart.action', stringify(params)).then(function(res){
					var data = res.data;

					if(data['success']) {
						vm.$message({
							message: '<%=rb.getString("ChengGong")%>',
							type: 'success'
						});

						vm.$refs.gnbStatusList.refresh();
					}else {
						vm.$message({
							message: data['message'],
							type: 'error'
						});
					}
				}).catch(function(){});

				vm.closeGnbBatchRetry();
				vm.gnbCancelViewSlide();
			},
			closeGnbBatchRetry(){
				var vm = this;

				vm.gnbBatchRetryForm.isContainSuccessTask = '0';
				vm.gnbShoBatchRetryShow = false;

				vm.gnbDeviceSelection = [];
				vm.$refs["gnbStatusList"].clearSelection();	
			},
			
			//GNB Execute status: start
			gnbStartOperStatus(row){
				var vm = this,
					params = {
						taskId: row.task_id,
					}
				axios.post("${ctx}/gnb/pnp/updatePnPManualPolicyTaskStart.action",stringify(params)).then(function(response){
					var data = response.data;
					if(data["success"]){
						vm.$refs.gnbStatusList.refresh();
					}else{
						vm.$message.error(data["message"]);
					}
				})
				vm.gnbCancelViewSlide();
			},
			//GNB Execute Status: delete
			gnbDeletStatus(row){
				var vm = this, params = {},

					url = '${ctx}/SON/SelfConfigurationTask/delSelfConfigTask.action';

				params = { taskId: row.task_id };

				vm.$confirm('<%=rb.getString("QueRenShanChu") %>','<%=rb.getString("QueRen") %>').then(function(r){
					if(r) {
						axios.post(url, stringify(params)).then(function(res){
							var data = res.data;

							if(data['success']) {
								vm.$message({
									message: '<%=rb.getString("ChengGong")%>',
									type: 'success'
								});

								vm.$refs.gnbStatusList.refresh();
							}else {
								vm.$message({
									message: data['message'],
									type: 'error'
								});
							}

							//清除已选
							vm.gnbDeviceSelection = [];
							vm.$refs["gnbStatusList"].clearSelection();
							vm.bulkSelectShow = false;

							vm.gnbCancelViewSlide();
						}).catch(function(){});
					}
				}).catch(function(){});
			},

			//-------------------------------------------------------------------GNB table operation
			gnbOptClick(row, evt) {
				var vm = this, modifyFlag, delFlag;
				//开关打开时， 不可修改，不可删除
				if(row.switch == '1'){
					modifyFlag = true;
					delFlag = true;
				}else{
					modifyFlag = false;
					delFlag = false;
				}

				vm.gnbMenus= [
					{label:'<%=rb.getString("XinXi")%>',cls:"el-icon el-icon-operation-info",code:'info', row: row},
					{label:'<%=rb.getString("XiuGai")%>',cls:"el-icon el-icon-operation-edit hidden CODE_PLUG_AND_PLAY",code:'modify', row: row, disable: modifyFlag},
					{label:'<%=rb.getString("JianChe")%>',cls:"el-icon el-icon-operation-scan hidden CODE_PLUG_AND_PLAY",code:'check', row: row},
					{label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete hidden CODE_PLUG_AND_PLAY",code:'del', row: row, disable: delFlag}
				];

		    	vm.$nextTick(function(){
		    		document.body.click();
					vm.$refs.gnbMenu.show(evt);
		    	});
			},
			//gnb policy menu click
			gnbMenuClick(row, evt) {
				var vm = this,
					code = row.code,
					data = row.row,
					actions = {
						info: vm.gnbView,
						modify: vm.gnbModify,
						del: vm.gnbDeleteData,
						check: vm.gnbCheckPolicy
					};

				if(actions[code]) {
					actions[code](data);
				}
			},

			//policy info
			gnbView(row) {
				var vm = this,
					rd = Math.random(),
					url = '${ctx}/gnb/pnp/toGNBAddConfig.action?r='+rd,
					params = { policyId: row.policyId };

				vm.slideURL = url;
				vm.slideHeader = false;
				vm.footerShow = false;
				vm.$refs.gnbSlider.showSlide(params,function(){
					eventBus.$emit('gnbInit-plugPlayConfig','readonly', row);
				});
				vm.gnbCancelViewSlide();
			},
			//policy gnbModify
			gnbModify(row){
				var vm = this, rd = Math.random(),
					url = '${ctx}/gnb/pnp/toGNBAddConfig.action?r='+rd,
					params = { policyId: row.policyId };

				vm.slideURL = url;
				vm.slideHeader = false;
				vm.footerShow = false;
				vm.$refs.gnbSlider.showSlide(params,function(){
					eventBus.$emit('gnbInit-plugPlayConfig','modify', row);
				});
				vm.gnbCancelViewSlide();
			},
			gnbCheckPolicy(row){
				var vm = this,
					params = {policyId: row.policyId};

				vm.gnbCkDLShow = true;
				if(vm.$refs.gnbSelectSnListForm){
				    vm.$refs.gnbSelectSnListForm.resetFields();
				}

				Object.assign(vm.gnbDetectParams, params);
				vm.gnbDetectSelections = [];
				vm.gnbDetectParams.searchText = '';
				vm.gnbDetectSearchText = '';
				vm.$nextTick(function(){
					vm.$refs.gnbDetectTb.clear();

					vm.gnbDetectURL = '${ctx}/gnb/pnp/queryPolicyGNBInfoPageList.action';
				})
			},
			gnbSendDetection() {
				var vm = this,
					params = {
						policyId: vm.gnbCurrentRow.policyId,
						smallCellCode: vm.gnbSelectSnListForm.serialNumber,
					};

				// 下发检测
				vm.$refs.gnbSelectSnListForm.validate((valid) => {
					if(valid){
						axios.post('${ctx}/SON/SelfConfiguration/exeSelfConfigDetectPolicy.action', stringify(params)).then(function(res){
							var data = res.data;
		
							if(data['success']) {
								vm.$message({
									message: '<%=rb.getString("ChengGong")%>',
									type:'success'
								});
		
								vm.gnbCkDLShow = false;
							}else {
								vm.$message.error(data['message']);
							}
						});
					}
				});
			},

			//gnb batch add 
			gnbAddBatchSnClick(){
				var vm = this;

				vm.gnbBatchSnDialog = true;
				if(vm.$refs.gnbBatchSnForm){
					vm.$refs.gnbBatchSnForm.resetFields();
				}
			},
			gnbSaveBatchSn(){
				var vm = this, 
					params = {}, 
					snStr = vm.gnbBatchSnForm.serialNumber,
					list = snStr.replace(/[(\r\n)\r\n\s；]+/g,';').split(';').filter(function(item){ return item.length > 0;});
									
				params.sns = list.join(",");
				params.policyId = vm.gnbCurrentRow.policyId;
				
				vm.$refs.gnbBatchSnForm.validate((valid) => {
					if(valid){
						axios.post('${ctx}/gnb/pnp/querySelectedPolicyGNBInfos.action', stringify(params)).then((res)=>{
							var data = res.data;
							
							if(data && data.length > 0){		
								vm.$refs.gnbDetectTb.appendCheckedRows(data);

								vm.gnbCloseBatchSn();
							}else{
								vm.$message('<%=rb.getString("MeiYouKePiPeiSheBei")%>')
							}
						})
					}
				})
			},
			gnbCloseBatchSn(){
				var vm = this;

				vm.gnbBatchSnDialog = false;
				vm.$refs.gnbBatchSnForm.resetFields();
			},

			gnbQueryDetect(val){
				var vm = this;
				vm.gnbDetectParams.searchText = vm.gnbDetectSearchText;
			},
			gnbSelectChange(selection) {
				var vm = this;

				vm.gnbDetectSelections = selection;
			},
			//policy delete
			gnbDeleteData(row) {
				var vm = this,
					url =  '${ctx}/gnb/pnp/delPnPPolicy.action',
					params = { 
						policyId: row.policyId,
						policyName: row.policyName
					};

				vm.$confirm('<%=rb.getString("QueRenShanChu") %>','<%=rb.getString("QueRen") %>').then(function(r){
					if(r) {
						axios.post(url,stringify(params)).then(function(res){
							var data = res.data;

							if(data['success']) {
								vm.$message({
									message: '<%=rb.getString("ChengGong")%>',
									type: 'success'
								});

								vm.$refs.gnbPolicyList.refresh();
							}else {
								vm.$message({
									message: data['message'],
									type: 'error'
								});
							}
						}).catch(function(){});
					}
				}).catch(function(){});
			},
			// table menu hide
			gnbHanderClose() {
				var vm = this;

				vm.$refs.gnbMenu.hide();
			},
			// Execute Status gnbRowClick view
			gnbCancelViewSlide(){
		    	var vm = this;
		    	vm.$refs.slideView.hide();
		    },

		    // slide close
		    gnbPlugPlayCancelSlide(){
		    	var vm = this;
				vm.$refs.gnbSlider.hide();
			},
			//----------------------------------------------------------- 已选 -----------------------------------------------------------
			//  status:2-进行中，3-未执行(等待) 表格操作项禁止点击 
			gnbIsDisabled(row,index){
				if(row.status == '2' || row.status == '3'){
					return false
				}else{
					return true
				}
			},
			/**
			* 列表选中
			* @param selection{Array}   选中数据
			*/
			gnbBatchSelect(selection){
				var vm = this;

				vm.gnbDeviceSelection = selection.map((item)=>{
					return Object.assign(item,{serial_number: item.serial_number});					
				});
			},
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

				vm.$refs["gnbStatusList"].clearSelection();	
				
				vm.bulkSelectShow = false;
			},
			// 设备已选表格 单个删除事件
			delBulkSelected(rows){
				//精简以下逻辑
				var vm = this, tabs = 'gnbStatusList', rowKey = 'task_id';

				vm.gnbDeviceSelection = vm.gnbDeviceSelection.filter((items)=>{
					return items[rowKey] != rows[rowKey]
				});
				var selection = this.$refs[tabs].$refs.ctableInner.store.states.selection,
					irow= selection.filter((items)=>{
						return items[rowKey] == rows[rowKey]
					})[0];
				vm.$refs[tabs].toggleRowSelection(irow,false);
				var idx = vm.$refs[tabs].ckList.indexOf(rows[rowKey]);
				vm.$refs[tabs].ckList.splice(idx,1);
				
				if(vm.gnbDeviceSelection.length == 0){
					vm.bulkSelectShow = false;
				}
			},
		},
		mounted() {
			this.gnbInit();
			eventBus.$off('gnbClose-config').$on('gnbClose-config', this.gnbCloseConfig);
			eventBus.$off('gnb_reload-config-list').$on('gnb_reload-config-list', this.gnbReloadConfig);

			eventBus.$off('gnbCancel-plugPlaySlide').$on('gnbCancel-plugPlaySlide',this.gnbPlugPlayCancelSlide);

			// 初始化拖拽
			var gnbHLine = document.querySelector('.horizontal-line-gnb');

			gnbPlugAddListener(gnbHLine,"mousedown",onmousedownVGnb);
		}
	});


	// 根据事件对象获取事件触发源对象
	function getTarget(evt){
		return evt.target || evt.srcElement;
	}
	/**
	* 绑定事件方法
	* @param element{dom}: 要绑定事件的对象
	* @param type{string}: 事件类型
	* @param listener{function}: 事件响应的方法
	* @param useCapture{boolean}: 是否在捕获阶段触发
	**/
	function gnbPlugAddListener(element,type,listener,useCapture){
		element.addEventListener?element.addEventListener(type,listener,useCapture):element.attachEvent("on" + type,listener);
	}

	/* 鼠标点击事件 */
	function onmousedownVGnb(event) {
		var _self = this;
		var lastY = event.clientY, d = document,
		rowItems = document.querySelectorAll('.flex-row-item-gnb'),
		preItem = rowItems[0],
		sufItem = rowItems[1];
		_self.clickDown = true;
		d.onmousemove = function(event){
			var evt = event || window.event;
			if(_self.clickDown == true) {
				var direction = evt.clientY - lastY;
				if(direction > 0) {//up
				setHeight(sufItem,direction);
				clearHeight(preItem);
				} else if(direction < 0){//down
				setHeight(preItem,-direction);
				clearHeight(sufItem);
				}
				lastY = evt.clientY;
			}
		};
		d.onmouseup = function(){
			d.onmousemove = null;
			d.onmouseup = null;
			_self.clickDown = false;
			$('#omc_app_ctn').resize();
		};

		function setHeight(item,offset) {
			var style = getComputedStyle(item),
				height = style.height.replace('px','') - offset;
			// maxHeight、minHeight 防止flex布局对高度计算的影响
			if(height < 300) height = 300;
			item.style.height = height + 'px';
			item.style.maxHeight = height + 'px';
			item.style.minHeight = height + 'px';
		}
		function clearHeight(item) {
			item.style.height = '100%';
			item.style.maxHeight = '';
			item.style.minHeight = '';
		}
	}
</script>
