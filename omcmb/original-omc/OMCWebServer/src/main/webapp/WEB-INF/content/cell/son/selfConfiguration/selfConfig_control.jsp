<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
	#selfConfigurationDiv{
		font-size:14px;
	}
	#selfConfigurationDiv .el-switch__core{
		height:17px;
	}
	#selfConfigurationDiv .el-switch__core:after{
		width:13px;
		height:13px;
	}
	.importPlanClass{
		display:flex;
		flex-direction:column;
		width:580px;
		height:400px;
		background:#fff;
		border:1px solid #DEDFE6;
		position:absolute;
		right:70px;
		top:-100px;
		border-radius:4px;
		box-shadow:0 2px 12px 0 rgba(0,0,0,0.1);
	}
	.importPlanClass .el-input{
		width:400px;
	}
	#selfConfigurationDiv .el-icon-status-disable:before{
		color:#C2C2C2;
	}
	.el-popover{
		word-break:break-all;
	}
	#selfConfigurationDiv .el-icon-circle-info:before{
		color:#CFCFCF;
	}
	#selfConfigurationDiv .el-icon-common-query-down:before{
		color:#363B4E;
	}
	.mapForm .el-form-item__label{
		line-height:28px;
	}
	.mapForm .importBox{
		margin-top:5px;
	}
	.mapFileItem{
		width:600px;
		height:200px;
		border:1px solid #DEDFE6;
		position:absolute;
		right:30px;
		top:-235px;
		background:#fff;
		box-shadow:rgb(0,0,0,0.16) 0px 0px 10px;
	}
	.modifyMapForm .el-form-item__label{
		line-height:28px;
	}
	/* .addTacFormBox .el-form-item__error { padding-top: 0;} */
	.enbIdItem .el-textarea__inner { width: 500px;}
	.enbIdTip { margin-left: 80px; font-size: 12px; color: #999999;}
</style>
<div id="selfConfigurationDiv" class="panelDefault">
	<el-tabs v-model="activeName" style="height:100%;" @tab-click="clickTab">
		<!-- 执行状态 -->
		<el-tab-pane name="exe" label="<%=rb.getString("ZhiXingZhuangTai")%>">
			<el-ctable id="exeStatusTable" ref="ctableExe" :url="exeUrl" :query-params="params_exe"  time="6"
			 :height="height" pagination="true" rownumber="true">
				
				<!-- 模糊查询 -- 执行状态 -->
				<template slot="toolbar">
					<div class="queryGroup">
						<el-input v-model="params_exe_form.searchText" @keyup.enter.native="queryExe" class='pairgrid-query' placeholder='<%=rb.getString("XiaoZhanBianMa")%>'></el-input>
						<i @click="queryExe" class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
					</div>
				</template>
				<el-table-column label="" width="30" prop="" class-name="no-text-tips">
					<template slot-scope="scope">
						<div class="el-icon el-icon-operation-more" @click="optClick(scope.row,event)" v-clickoutside="handerClose" style="cursor: pointer;"></div>
					</template>
				</el-table-column>
				<el-table-column label="<%=rb.getString("XiaoZhanBianMa")%>" prop="serial_number"></el-table-column>
				<el-table-column label="<%=rb.getString("YuanECI")%>" prop="originECI"></el-table-column>
				<el-table-column label="<%=rb.getString("MuBiaoECI")%>" prop="targetECI"></el-table-column>
				<el-table-column label="<%=rb.getString("ZiKaiZhanZhiXingFangShi") %>" prop="execute_type" :formatter="exeTypeConfigFmt"></el-table-column>
				<el-table-column label="<%=rb.getString("JinDu")%>" prop="execute_procedure"></el-table-column>
				<el-table-column label="<%=rb.getString("ZhuangTai")%>" prop="status">
					<template slot-scope="scope">
						<div v-if="scope.row.status == '0'">
							<span class='el-icon el-icon-status-waiting1' style='margin-right:5px;'></span><%=rb.getString("DengDai")%>
						</div>
						<div v-if="scope.row.status == '1'">
							<span class='el-icon el-icon-status-inProgress' style='margin-right:5px;'></span><%=rb.getString("JinXingZhong")%>
						</div>
						<div v-if="scope.row.status == '2'">
							<span class='el-icon el-icon-status-terminate' style='margin-right:5px;'></span><%=rb.getString("YiJieShu")%>
						</div>
						<div v-if="scope.row.status == '3'">
							<span class='el-icon el-icon-status-waiting1' style='margin-right:5px;'></span><%=rb.getString("DaiZhiXing")%>
						</div>
					</template>
				</el-table-column>
				<el-table-column label="<%=rb.getString("JieGuo")%>" prop="result" :formatter="resultConfigFmt"></el-table-column>
				<el-table-column label="<%=rb.getString("KaiShiShiJian")%>" prop="start_time"></el-table-column>
				<el-table-column label="<%=rb.getString("JieShuShiJian")%>" prop="end_time"></el-table-column>
			</el-ctable>
			<el-cmenu ref="menu_exe" :data="menus_exe" @click="clickMenu"></el-cmenu>
		</el-tab-pane>
		<!-- self start -->
		<el-tab-pane name="selfStart" label="Self-start" style="overflow:auto">
			<div style="margin-left:60px;margin-top:30px;">
				<p><span><%=rb.getString("ShiFouQiYong") %></span><el-switch style='margin-left:10px;' v-model="selfForm.selfStartEnable" active-color="#4D84FF" inactive-color="#BDC1C6" @change="changeSwitch($event,'main')" :disabled="openAllFlag"></el-switch></p>
			</div>
			<div style="flex: 1 1 auto;overflow:auto">
				<div style='margin-left:60px;margin-top:20px;'>
					<div style='margin-top:20px;'>
						<label><%=rb.getString("ZhiXingFangShi") %></label>
						<div style="margin-top:5px;width:600px;height:25px;border:1px solid #DEDFE6;padding-top:15px;">
							<el-radio-group v-model="selfForm.executeType" style='margin-left:15px;' :disabled="openStartFlag" @change="changeSwitch($event,'exetype')">
								<el-radio label="0"><%=rb.getString("ZiDongZhiXingConfig") %></el-radio>
								<el-radio label="1" style='margin-left:135px;'><%=rb.getString("ShouDongZhiXingConfig") %></el-radio>
							</el-radio-group>
						</div>
					</div>
				</div>
				<p style="height:2px;background:#E9E9E9;margin-top:40px;margin-left:20px;"></p>
				
				<!-- 软件升级 -->
				
				<div class="group-title not-extend" style='margin-left:35px;margin-top:40px;'>
					<span class="title-icon"></span>
					<span class="title-text"><%=rb.getString("RuanJianShengJi") %></span>
					<el-switch v-model="selfForm.upgradeEnable" active-color="#4D84FF" inactive-color="#BDC1C6" :disabled="openStartFlag" @change="changeSwitch($event,'upgrade')"></el-switch>
				</div>
				<div style='margin-left:60px;margin-top:15px;width:93%;'>
					<div style="display:flex">
						<label style="flex:1 1 auto;line-height:30px;"><%=rb.getString("RuanJianLieBiao") %></label>
						<span v-show="showOpenStart" @click="addUpgrade" class='el-icon el-icon-circle-add' style='font-size:24px;'></span>
					</div>
					<div style="width:100%;height:300px;border:1px solid #DEDFE6;">
						<el-ctable	ref="ctableUpgrade" :url="upgradeUrl" :query-params="params_upgrade" :row-key="'id'"
						:height="height" pagination="true" rownumber="true">
							
							<!-- 模糊查询 -- 软件升级 -->
							<template slot="toolbar">
								<div class="queryGroup">
									<el-input v-model="params_upgrade_form.searchText" @keyup.enter.native="queryUpgrade" class='pairgrid-query' placeholder='<%=rb.getString("ChuShiBanBen")%> / <%=rb.getString("MuBiaoBanBen")%>'></el-input>
									<i @click="queryUpgrade" class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
								</div>
							</template>
							<el-table-column label="" prop="" width="30" class-name="no-text-tips" v-if="tableSelectShow">
								<template slot-scope="scope">
									<div class="el-icon el-icon-operation-more" :class="openStartClass" @click="optClickUpgrade(scope.row,event)" v-clickoutside="handerCloseUpgrade" style="cursor: pointer;"></div>
								</template>
							</el-table-column>
							<el-table-column label="<%=rb.getString("ZhuangTai")%>" prop="status">
								<template slot-scope="scope">
									<div v-if="scope.row.status == '1'">
										<span class="el-icon el-icon-status-enable" style='margin-right:5px;'></span><span><%=rb.getString("QiYong")%></span>
									</div>
									<div v-else>
										<span class="el-icon el-icon-status-disable"></span><span style='margin-left:5px;color:#C2C2C2'><%=rb.getString("JinYong")%></span>
									</div>
								</template>
							</el-table-column>
							<el-table-column label="<%=rb.getString("ChanPinLeiXingBiaoZhi")%>" prop="product_value"></el-table-column>
							<el-table-column label="<%=rb.getString("ChuShiBanBen")%>" prop="original_version">
								<template slot-scope='scope'>
									<div v-if='scope.row.original_version == null'></div>
									<div v-else-if='scope.row.original_version.split(",").length == 1'>{{scope.row.original_version}}</div>
									<div v-else>
										<span>{{scope.row.original_version.split(',')[0]}}...</span>
										<el-popover style='word-wrap:break-word' placement='bottom' trigger='click' width='200'>
											<span v-html="scope.row.original_version.replace(/,/g,',<br>')"></span>
											<span slot='reference' style='color:#7584FF;cursor:pointer'>[{{scope.row.original_version.split(',').length}}]</span>
										</el-popover>
									</div>
								</template>
							</el-table-column>
							<el-table-column label="<%=rb.getString("MuBiaoBanBen")%>" prop="dest_version"></el-table-column>
							<el-table-column label="<%=rb.getString("BaoLiuPeiZhi")%>" prop="preserve_setting" :formatter="retainConfigFmt"></el-table-column>
						</el-ctable>
						<el-cmenu ref="menu_upgrade" :data="menus_upgrade" @click="clickMenuUpgrade"></el-cmenu>
					</div>
				</div>
				<p style="height:2px;background:#E9E9E9;margin-top:40px;margin-left:20px;"></p>
				
				<!-- license -->
				
				<div class="group-title not-extend" style='margin-left:35px;margin-top:40px;'>
					<span class="title-icon"></span>
					<span class="title-text"><%=rb.getString("LicenseXiaFa") %></span>
					<el-switch v-model="selfForm.licenseEnable" active-color="#4D84FF" inactive-color="#BDC1C6" :disabled="openStartFlag" @change="changeSwitch($event,'license')"></el-switch>
				</div>
				<div style="font-size:12px;margin-left:60px;margin-top:15px;">
					<span class="el-icon el-icon-circle-info" style="font-size:14px"></span>
					<span @click="viewLicense" style="color:#4D84FF;text-decoration:underline;cursor:pointer;"><%=rb.getString("DianJiChaKanLicense") %></span>
				</div>
				<p style="height:2px;background:#E9E9E9;margin-top:40px;margin-left:20px;"></p>
				
				<!-- 参数配置 -->
				
				<div class="group-title not-extend" style='margin-left:35px;margin-top:40px;'>
					<span class="title-icon"></span>
					<span class="title-text"><%=rb.getString("CanShuZiPeiZhi") %></span>
					<el-switch v-model="selfForm.selfConfigEnable" active-color="#4D84FF" inactive-color="#BDC1C6" :disabled="openStartFlag" @change="changeSelfConfig($event,'config')"></el-switch>
				</div>
				<div style="margin-left:60px;margin-top:15px;width:93%;position:relative">
					<p>
						<span class="el-icon el-icon-circle-info" style="font-size:14px;"></span>
						<span style="color:#BBB"><%=rb.getString("CanShuPeiZhiTiShi") %></span>
					</p>
					<div style="display:flex;margin-top:15px;">
						<label style="flex:1 1 auto;line-height:30px;"><%=rb.getString("ZhiDingCanShu")%></label>
						<p style="display:inline-block;margin-bottom:5px;">
							<span v-show="showOpenStart" @click="importConfigPlan" class='el-icon el-icon-circle-import' style='font-size:24px;margin-right:10px;'></span>
							<span @click="exportConfigPlan" class='el-icon el-icon-circle-export' style='font-size:24px;'></span>
						</p>
					</div>
					<div style="height:300px;border:1px solid #DEDFE6;">
						<el-ctable	ref="ctablePlan" :url="urlConfigPlan" :query-params="params_plan" :row-key="'id'"
						:height="height" pagination="true" rownumber="true">
							
							<!-- 模糊查询 -- Configuration Plan -->
							<template slot="toolbar">
								<div class="queryGroup">
									<el-input v-model="params_plan_form.searchText" @keyup.enter.native="queryPlan" class='pairgrid-query' placeholder='<%=rb.getString("XiaoZhanBianMa")%>'></el-input>
									<i @click="queryPlan" class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
								</div>
							</template>
							<el-table-column label="" prop="" width="30" class-name="no-text-tips">
								<template slot-scope="scope">
									<div class="el-icon el-icon-operation-more" :class="openStartClass" @click="optClickPlan(scope.row,event)" v-clickoutside="handerClosePlan" style="cursor: pointer;"></div>
								</template>
							</el-table-column>
							<el-table-column label="<%=rb.getString("XiaoZhanBianMa")%>" prop="serial_number" width="200"></el-table-column>
							<el-table-column label="<%=rb.getString("HostName")%>" prop="host_name" width="100"></el-table-column>
							<el-table-column label="<%=rb.getString("ZhiChiPinDuan")%>" prop="bands_support" width="150"></el-table-column>
							<el-table-column label="<%=rb.getString("DaiKuan")%>(MHz)" width="150" prop="band_width" :formatter="bandWidthFmt"></el-table-column>
							<el-table-column label="<%=rb.getString("PinDian")%>" prop="frequency" :formatter="earfcnFmt" width="150"></el-table-column>
							<el-table-column label="<%=rb.getString("ZiZhenPeiBi")%>" prop="subframe_assignment" :formatter="sfassignmentFmt" width="150"></el-table-column>
							<el-table-column label="<%=rb.getString("TeShuZiZhenPeiBi")%>" prop="special_subframe_patterns" width="150"></el-table-column>
							<el-table-column label="<%=rb.getString("PLMN")%>" prop="plmn_id"></el-table-column>
							<el-table-column label="<%=rb.getString("TAC")%>" prop="tac"></el-table-column>
							<el-table-column label="ECI" prop="cell_identity"></el-table-column>
							<el-table-column label="<%=rb.getString("PCI")%>" prop="phycellid"></el-table-column>
							<el-table-column label="<%=rb.getString("GenXuLieSuoYin")%>" prop="root_sequence_index" width="150"></el-table-column>
							<el-table-column label="<%=rb.getString("GengXinRen")%>" prop="uploader"></el-table-column>
							<el-table-column label="<%=rb.getString("GengXinShiJian")%>" prop="update_time" width="150"></el-table-column>
						</el-ctable>
						<el-cmenu ref="menu_configPlan" :data="menus_configPlan" @click="clickMenuPlan"></el-cmenu>
					</div>
					
					<!-- Auto Map -->
					
					<div style="display:flex;margin:15px 0 10px; position: relative;">
						<div style='display: flex;'>
							<label style="flex:1 1 auto;line-height:20px;margin-right:20px;">TAC Auto Configure</label>
							<el-switch v-model="selfForm.selfAutoMapEnable" active-color="#4D84FF" inactive-color="#BDC1C6" :disabled="openStartFlag" @change="changeSwitch($event,'automap')"></el-switch>
							<!-- MAP - 0, TAC - 1 -->
							<el-radio-group v-model="selfForm.autoConfigMode" style='margin-left: 40px; margin-top: 3px;' :disabled="openStartFlag" @change="changeSwitch($event,'autoTac')">
								<el-radio label="0">Coordinate-TAC Mapping Mode</el-radio>
								<el-radio label="1">eNB ID-TAC Mapping Mode</el-radio>
							</el-radio-group>
							<div style='margin-left: 20px;' v-show="selfForm.autoConfigMode == '1'" >
				                <el-checkbox v-model="selfForm.defaultTac" true-label="1" false-label="0" :disabled="openStartFlag" @change="changeSwitch($event,'checkTac')"></el-checkbox>
				                <span style='margin-left: 10px;'><%=rb.getString("TACPeiZhiTiShi")%></span>
				            </div>
						</div>
						<p style="position: absolute; right: 0; top: 0;" v-show="showOpenStart">
							<span v-show="selfForm.autoConfigMode == '1'" @click="addMapTac" class='el-icon el-icon-circle-add' style='font-size:24px;margin-right:10px;'></span>
							<span v-show="selfForm.autoConfigMode == '0'" @click="importMapFile" class='el-icon el-icon-circle-import' style='font-size:24px;margin-right:10px;'></span>
							<span v-show="selfForm.autoConfigMode == '0'" @click="clearMapFile" class='el-icon el-icon-circle-clear' style='font-size:24px;'></span>
						</p>
					</div>
					<div style="height:300px;border:1px solid #DEDFE6;position:relative">
						<el-ctable ref="enbIdTactable" :url='enbIdUrl' :row-key="'id'" v-show="selfForm.autoConfigMode == '1'"
						:height="height" pagination="true" rownumber="true">
							<el-table-column label="" prop="" width="30" class-name="no-text-tips">
								<template slot-scope="scope">
									<div class="el-icon el-icon-operation-more" :class="openStartClass" @click="enbIdTacOptClick(scope.row,event)" v-clickoutside="enbIdTachanderClose" style="cursor: pointer;"></div>
								</template>
							</el-table-column>
							<el-table-column label="TAC" prop="tac"></el-table-column>							
							<el-table-column label="eNB ID" prop="eciRange" :show-overflow-tooltip="true"></el-table-column>
						</el-ctable>
						<el-cmenu ref="menu_enbIdtac" :data="menus_enbIdtac" @click="enbIdTacClickMenu"></el-cmenu>
						
						<el-ctable ref="ctableMap" :url="urlMap" :row-key="'id'" v-show="selfForm.autoConfigMode == '0'"
						:height="height" pagination="true" rownumber="true">
							<el-table-column label="" prop="" width="30" class-name="no-text-tips">
								<template slot-scope="scope">
									<div class="el-icon el-icon-operation-more" :class="openStartClass" @click="optClickMap(scope.row,event)" v-clickoutside="handerCloseMap" style="cursor: pointer;"></div>
								</template>
							</el-table-column>
							<el-table-column label="<%=rb.getString("SheBeiZuMingCheng")%>" prop="groupName"></el-table-column>
							<el-table-column label="TAC" prop="tac"></el-table-column>
							<el-table-column label="ECI Range" prop="eciRange"></el-table-column>
							<el-table-column v-if="false" label="<%=rb.getString("ZuoBiao")%>" prop="coordinates">
								<template slot-scope="scope">
									<div v-if="scope.row.coordinates == ''">
										<span>No</span>
									</div>
									<div v-else>
										<span>Yes({{scope.row.coordinates}}...)</span>
									</div>
								</template>
							</el-table-column>
						</el-ctable>
						<el-cmenu ref="menu_map" :data="menus_map" @click="clickMenuMap"></el-cmenu>
						<!-- 导入文件悬浮框 -->
						<div class='mapFileItem' v-show="showMapImport" v-load="mapLoading">
							<div class="el-card__header">
								<span ><%=rb.getString("DaoRu")%></span>
								<span class=" el-icon el-icon-close" style="font-size:16px;" @click="closeMapImport"></span>
							</div>
							<div class="el-card__body">
								<el-form ref="mapForm" :model="mapForm" :rules="mapRule" label-position="left" label-width="150" class='mapForm'>
									<el-upload
											ref="upload_polygon"
											:before-upload='beforeUploadPolygon' 
											:show-file-list=false 
											:auto-upload="false"
											:on-change="fileChangePolygon"  
											action=""
										>
									<el-form-item label="TAC-POLYGON File" prop="file_polygon">
											<el-input style="width:250px;" class="uploadInput" :readonly="true" :value='mapForm.file_polygon' placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>'>
												<a slot="suffix" class="el-icon el-icon-operation-import importBox" @click="selectPolFile"></a>
											</el-input>
											<span style="margin-left:10px;color:#bbb">Supported .KML</span>								
									</el-form-item>
									<a slot="trigger" ref="file_polygon"></a>
									</el-upload>
									<p style="color:#BBB;font-size:12px;margin-left:150px;">
										<span style='font-size:14px;' class="el-icon el-icon-circle-info"></span>
										<%=rb.getString("KMLFangShiZhuiJia")%>
									</p>
								</el-form>
							</div>
							<div class="el-card__footer">
								<el-button @click="submitMap" type="primary"><%=rb.getString("QueDing")%></el-button>
								<el-button @click="closeMapImport"><%=rb.getString("QuXiao")%></el-button>
							</div>
						</div>
					</div>
					<p style="margin-top:20px;"><%=rb.getString("ZiDongFenPeiCanShu")%></p>
					<div style="margin-top:5px;height:300px;border:1px solid #DEDFE6;margin-bottom:20px;">
						<el-ctable	ref="ctableRule" :url="urlConfigRule" :query-params="params_rule" :row-key="'id'"
						:height="height" pagination="true" rownumber="true">
							
							<!-- 模糊查询 -- Configuration Rule -->
							<!-- <template slot="toolbar">
								<div class="queryGroup">
									<el-input class='pairgrid-query' placeholder='Serial Number'></el-input>
									<i class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
								</div>
							</template> -->
							<el-table-column label="" prop="" width="30" class-name="no-text-tips">
								<template slot-scope="scope">
									<div class="el-icon el-icon-operation-more" :class="openStartClass" @click="optClickRule(scope.row,event)" v-clickoutside="handerCloseRule" style="cursor: pointer;"></div>
								</template>
							</el-table-column>
							<el-table-column label="<%=rb.getString("ZhuangTai")%>" prop="status">
								<template slot-scope="scope">
									<div v-if="scope.row.switch_enable == '1'">
										<span class="el-icon el-icon-status-enable" style='margin-right:5px;'></span><span><%=rb.getString("QiYong")%></span>
									</div>
									<div v-else>
										<span class="el-icon el-icon-status-disable"></span><span style='margin-left:5px;color:#C2C2C2'><%=rb.getString("JinYong")%></span>
									</div>
								</template>
							</el-table-column>
							<el-table-column label="<%=rb.getString("ChanPinLeiXingBiaoZhi")%>" prop="platform" width="120"></el-table-column>
							<el-table-column label="<%=rb.getString("ZhiChiPinDuan")%>" prop="bands_support" width="150"></el-table-column>
							<el-table-column label="<%=rb.getString("DaiKuan")%>(MHz)" width="150" prop="band_width" :formatter="bandWidthFmt"></el-table-column>
							<el-table-column label="<%=rb.getString("PinDian")%>" prop="frequency" :formatter="earfcnFmt" width="150"></el-table-column>
							<el-table-column label="<%=rb.getString("ZiZhenPeiBi")%>" prop="subframe_assignment" :formatter="sfassignmentFmt" width="150"></el-table-column>
							<el-table-column label="<%=rb.getString("TeShuZiZhenPeiBi")%>" prop="special_subframe_patterns" width="170"></el-table-column>
							<el-table-column label="<%=rb.getString("PLMN")%>" prop="plmn_id"></el-table-column>
							<el-table-column label="<%=rb.getString("TAC")%>" prop="tac"></el-table-column>
							<el-table-column label="ECI" prop="cell_identity"></el-table-column>
							<el-table-column label="<%=rb.getString("PCI")%>" prop="phycellid"></el-table-column>
							<el-table-column label="<%=rb.getString("GenXuLieSuoYin")%>" prop="root_sequence_index" width="150"></el-table-column>
						</el-ctable>
						<el-cmenu ref="menu_configRule" :data="menus_configRule" @click="clickMenuRule"></el-cmenu>
					</div>
					<!-- 导入悬浮框 -->
					<div class="importPlanClass" v-show="importVisible" v-loading="importLoading">
						<div class="el-card__header">
							<span ><%=rb.getString("DaoRu")%></span>
							<span @click="cancelImportPlan" class=" el-icon el-icon-close" style="font-size:16px;"></span>
						</div>
						<div class="el-card__body" style="background:#fff;flex:1 1 auto">
							<el-form ref="importForm" :model="importForm" :rules="importRules" label-position="top" style="margin-left:30px;margin-top:30px;">
								<el-form-item label="<%=rb.getString("DaoRuLeiXing")%>">
									<div style="width:400px;height:25px;border:1px solid #DEDFE6;padding-top:15px;">
										<el-radio-group v-model="importForm.importType">
											<el-radio label="0" style="margin-left:20px;margin-right:50px;"><%=rb.getString("ZhuiJia")%></el-radio>
											<el-radio label="1"><%=rb.getString("FuGai")%></el-radio>
										</el-radio-group>
									</div>
								</el-form-item>
								<el-form-item label="<%=rb.getString("WenJian")%>" prop="filePath">
									<el-input :disabled="true" style='width:400px;' v-model="importForm.filePath">
										<i @click="importFile" slot='suffix' style='display:inline-block;width:28px;height:28px;margin:4px -9px 0 0;' class='el-icon el-icon-operation-import'></i>
									</el-input>
								</el-form-item>
								<p style="color:#BBB"><span style='font-size:14px;' class="el-icon el-icon-circle-info"></span><%=rb.getString("DaoRuWenJianTiShi")%></p>
								<div style="position:relative;margin-top:10px;">
									<p style="cursor:pointer;max-width:160px;" @click="showExportType" v-clickoutside="handerCloseExport">
										<span class='el-icon el-icon-common-download'></span>
										<span style='text-decoration:underline;cursor:pointe;font-size:14px;'><%=rb.getString("DaoChuMuBan")%></span>
										<span class="el-icon el-icon-common-query-down"></span>
									</p>
									<div class="cmenu item-left" v-show="exportTypeFlag" style="position:absolute;left:150px;bottom:-30px;">
										<div v-for="item in exportTypeOptions" class="cmenu-item" @click="exportTemp(item.id)">
											<span>{{item.text}}</span>
										</div>
									</div>
								</div>
								<div style='margin-top:45px;'>
									<el-button @click="saveImportFile" type="primary"><%=rb.getString("QueDing")%></el-button>
									<el-button @click="cancelImportPlan"><%=rb.getString("QuXiao")%></el-button>
								</div>
							</el-form>
						</div>
					</div>
				</div>
			</div>
		</el-tab-pane>
	</el-tabs>
	<el-slide ref="slide" :url="slideUrl" :title="slideTitle" :footer="slideFooter" :header='slideHeader' :position="slidePosition"
	 :height="slideHeight" :modal='slideModal'  :width="slideWidth" :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'"  @cancel='cancelSlide' @ok='saveSlide'>
	 	
	 </el-slide>
	 <el-dialog  title='<%=rb.getString("ChaKan")%> License' width='800px' :visible.sync='licenseVisible' :close-on-click-modal="false">
		<div style="height:430px;">
			<el-ctable	ref="ctableUpgrade" :url="licenseUrl" :query-params="params_license" :row-key="'id'"
				:height="height" pagination="true" rownumber="true">
								
				<!-- 模糊查询 -- 软件升级 -->
				<template slot="toolbar">
					<div class="queryGroup">
						<el-input v-model="params_license_form.search_text" @keyup.enter.native="queryLicense" class='pairgrid-query' placeholder='<%=rb.getString("XiaoZhanBianMa")%>'></el-input>
						<i @click="queryLicense" class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
					</div>
				</template>
				<el-table-column prop="connection_status" width="50">
					<template slot-scope="scope">
						<div :class="{
							'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
							'':scope.row.have_connected==2,
							'conn_exc':scope.row.connection_status=='Exception',
							'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
					</template>
				</el-table-column>
				<el-table-column label="<%=rb.getString("XiaoZhanBianMa")%>" prop="serial_number"></el-table-column>
				<el-table-column label="<%=rb.getString("LicenseWenJian")%>" prop="file_name"></el-table-column>
				<el-table-column label="<%=rb.getString("ShangChuanShiJian")%>" prop="upload_time"></el-table-column>
				<el-table-column label="<%=rb.getString("ZhuangTai")%>" prop="execute_status" :formatter="executeStatus"></el-table-column>
			</el-ctable>
		</div>
	</el-dialog>
	<el-dialog title='<%=rb.getString("XinXi")%>' width='400px' :visible.sync='downloadVisible' :close-on-click-modal="false">
		<div>
			<p style="color:#797979;font-size:14px;"><%=rb.getString("BuFenShuJuCuoWuQingChongXingBianJi")%></p>
			<el-button style="margin-top:20px;margin-left:250px;" @click="downloadErrorFile" type="primary"><%=rb.getString("XiaZai")%></el-button>
		</div>
	</el-dialog>
	<el-dialog  title='Modify' width='500px' :visible.sync='modifyMapVisible' :close-on-click-modal="false">
		<el-form ref="modifyMapForm" :model="modifyMapForm" :rules="editMapRule" label-position="left" label-width="120" class="modifyMapForm">
			<el-form-item label="<%=rb.getString("SheBeiZuMingCheng")%>" prop="groupName">
				<el-input v-model="modifyMapForm.groupName"></el-input>
			</el-form-item>
			<el-form-item label="TAC" prop="tac">
				<el-input v-model="modifyMapForm.tac"></el-input>
			</el-form-item>
			<el-form-item label="ECI Range" prop="idRange">
				<el-input v-model="idRangeStart" style="width:100px;"></el-input> --
				<el-input v-model="idRangeEnd" style="width:100px;"></el-input>
			</el-form-item>
		</el-form>
		<div style='margin-top:45px;'>
			<el-button @click='editMapSubmit' type="primary"><%=rb.getString("QueDing")%></el-button>
			<el-button @click='cancelEditMap'><%=rb.getString("QuXiao")%></el-button>
		</div>
	</el-dialog>
	<!-- eNB ID - TAC : add -->
	<el-dialog :title='AddModifyTacTitle' width='700px' :visible.sync='addTacVisible' :close-on-click-modal="false">
		<el-form ref="addTacForm" :model="addTacForm" :rules="addTacRule" label-position="left" label-width="80" class="modifyMapForm addTacFormBox">			
			<el-form-item label="TAC" prop="tac">
				<el-input v-model="addTacForm.tac" :disabled='enbIdTacDisabled'></el-input>
			</el-form-item>
			<el-form-item label="eNB ID" prop="eciRange" class='enbIdItem' style='margin-bottom: 2px;'>
				<el-input type="textarea" v-model="addTacForm.eciRange"></el-input>
			</el-form-item>
			<span class='enbIdTip' v-show='enbIdTip'><%=rb.getString("eNBIDShuRuTiShi")%></span>
		</el-form>
		<div style='margin-top:45px;'>
			<el-button @click='addTacSubmit' type="primary"><%=rb.getString("QueDing")%></el-button>
			<el-button @click='addTacCancel'><%=rb.getString("QuXiao")%></el-button>
		</div>
	</el-dialog>
</div>
<%-- 表单-上传基站列表文件 --%>
<form enctype="multipart/form-data" method="post" id="uploadForm_configPlan" style="display: none;">
	<input name="fileSize"  value="" hidden="true">
    <input name="uploadFile" type="file" id="uploadFileConfigPlan">
    <input name="operType" value="">
</form>
<script>
	var selfVue = new Vue({
		el:"#selfConfigurationDiv",
		data(){
			var vm = this;
			var validateFilePath = (rule,value,callback) => {
				if(value == ""){
					callback(new Error("<%=rb.getString("QingXianXuanZeWenJian")%>"))
				}else if(!fileFormatMatch(value,"xlsx,xls,csv")){
					callback(new Error("<%=rb.getString("DaoRuWenJianGeShi")%>"))
				}else{
					callback();
				}
			};
			var filePolygonValidate = (rule,value,callback) => {
				if(value) {
					if(value.length>100) {
						callback('<%=rb.getString("WenJianMingBuNengChaoGuoYaoQiu")%>');
					}else if(!fileFormatMatch(value,"KML")){
						callback('<%=rb.getString("GeShiCuoWu")%>');
					}else {
						callback();
					}
				}else {
					callback('<%=rb.getString("QingXianXuanZeWenJian")%>');
				}
			};
			var fileGroupValidate = (rule,value,callback) => {
				if(value) {
					if(value.length>100) {
						callback('<%=rb.getString("WenJianMingBuNengChaoGuoYaoQiu")%>');
					}else if(!fileFormatMatch(value,"xlsx,xls,csv")){
						callback('<%=rb.getString("GeShiCuoWu")%>');
					}else {
						callback();
					}
				}else {
					callback('<%=rb.getString("QingXianXuanZeWenJian")%>');
				}
			};
			var validateTac = (rule,value,callback) => {
				var reg = /^(\d+)$/;
				var minVal = rule.min;
				var maxVal = rule.max;
				var tipStr = '<%=rb.getString("ZhengXing")%> <%=rb.getString("DouHao")%> '+minVal+' - '+maxVal;
				var msgStr = tipStr;
				var showFlag = false;
				if(value == ""){
					showFlag = true;
				}else{
					var showFlag = false;
					if (!(reg.test(value))) {
						showFlag = true;
					}else{
						
						if(minVal-value>0){
							showFlag = true;
						}
						if(maxVal-value<0){
							showFlag = true;
						}
						
					}
				}
				if(showFlag){
					callback(new Error(msgStr));
				}else{
					callback();
				}
			};
			var validateIDRange = (rule,value,callback) => {
				var reg = /^[0-9]*$/;
				
				if (vm.idRangeStart=="" && vm.idRangeEnd ==""){
					callback();
				}else if (vm.idRangeStart && vm.idRangeEnd){
					if(reg.test(vm.idRangeStart) && reg.test(vm.idRangeEnd)){
						if (parseInt(vm.idRangeStart) > parseInt(vm.idRangeEnd)){
							callback('The ID Range is incorrect.');
						}else{
							callback();
						}
					}else {
						callback('The ID Range is incorrect.');
					}
					
				}else {
					callback('The ID Range is incorrect.');
				}

			},
			validateEnbId = function(rule,value,callback) { 				
				var curIdReg = /[^\d,]/g, curArr = value.split(',');
				
				if(value != null && value.length != 0 ){
					var enbIdFlag = curArr.every(function(item,index){
						return (!curIdReg.test(item) && (parseInt(item)>=0 && parseInt(item) <= 9999999)) 
					})
					if(enbIdFlag){
						vm.enbIdTip = true;
						callback();
					}else{
						vm.enbIdTip = false;
						callback(new Error('<%=rb.getString("eNBIDShuRuTiShi")%> <%=rb.getString("FanWei")%>:0-9999999'));
					}
				}else if (value == null || value.length == 0) {
					vm.enbIdTip = false;
					callback(new Error('<%=rb.getString("eNBIDShuRuTiShi")%>'))
				}else{
					vm.enbIdTip = true;
					callback();
				}
			};
			return{
				activeName:"exe",
				exeUrl:"${ctx}/SON/SelfConfigurationTask/getSelfConfigTaskPageList.action",
				params_exe:{
					timeZone:timeZone,
					searchText:""
				},
				params_exe_form:{
					searchText:""
				},
				height:"100%",
				menus_exe:[],
				rowDataExe:[],
				selfForm:{
					selfStartEnable:false,
					executeType:"0",
					upgradeEnable:false,
					licenseEnable:false,
					selfConfigEnable:false,
					selfAutoMapEnable:false,
					autoConfigMode: '0',
					defaultTac: '0',
				},
				upgradeUrl:"${ctx}/SON/SelfConfiguration/getSelfUpgradePlansPageList.action",
				params_upgrade:{
					timeZone:timeZone,
					searchText:""
				},
				params_upgrade_form:{
					searchText:""
				},
				menus_upgrade:[],
				menus_configPlan:[],
				menus_configRule:[],
				slideUrl:"",
				slideTitle:"",
				slideFooter:"",
				slideHeader:"",
				slidePosition:"",
				slideHeight:"",
				slideWidth:"",
				slideModal:"",
				licenseVisible:false,
				importForm:{
					importType:"0",
					filePath:""
				},
				importRules:{
					filePath:[
						{validator:validateFilePath}
					]
				},
				// exportTypeOptions:[{id:"intel",text:"R Series"},{id:"qa",text:"Q Series"},{id:'NBIOT',text:"NB Series"},{id:'BBU_XSS',text:"GNB Series"}],
				exportTypeOptions:[{id:"intel",text:"R Series"},{id:"qa",text:"Q Series"},{id:'NBIOT',text:"NB Series"}],
				exportTypeFlag:false,
				importVisible:false,
				rowDataPlan:[],
				rowDataRule:[],
				rowDataUpgrade:[],
				urlConfigPlan:"${ctx}/SON/SelfConfiguration/getSelfConfigPlanningPageList.action",
				params_plan:{
					timeZone:timeZone,
					searchText:""
				},
				params_plan_form:{searchText:""},
				urlConfigRule:"${ctx}/SON/SelfConfiguration/getSelfProductRulesPageList.action",
				params_rule:{timeZone:timeZone},
				operTypePlan:"",
				operTypeRule:"",
				operTypeUpgrade:"",
				configType:"",
				licenseUrl:"${ctx}/cell/license/getAllLicenseInfoData.action",
				params_license:{
					timeZone:timeZone,
					search_text:"",
					isGnb: '0'
					// isGnb: 'all'
				},
				params_license_form:{search_text:""},
				openStartFlag:false,
				showOpenStart:true,
				openStartClass:"",
				downloadVisible:false,
				importLoading:false,
				openAllFlag:false,
				mapForm:{
					file_polygon:''
				},
				mapRule:{
					file_polygon:[
						{validator: filePolygonValidate}
					]
				},
				showMapImport:false,
				file_polygon:'',
				urlMap:"${ctx}/SON/SelfConfiguration/querySelfCoordinateTacPageList.action",
				rowDataMap:[],
				menus_map:[],
				modifyMapVisible:false,
				modifyMapForm:{
					groupName:'',
					tac:'',
					idRange:''
				},
				idRangeStart:'',
				idRangeEnd:'',
				editMapRule:{
					tac:[{validator:validateTac,min:0,max:65535,trigger:'blur'}],
					idRange:[{validator:validateIDRange,trigger:'blur'}]
				},
				mapLoading:false,
				
				//new
				enbIdUrl: '${ctx}/SON/SelfConfiguration/querySelfConfigTacPageList.action',
				addTacVisible: false,
				addTacForm: {
					tac: '',
					eciRange:"",
				},
				
				addTacRule: {
					tac: [{validator: validateTac, min: 0, max: 65535, trigger: 'blur'}],
					eciRange: [{validator: validateEnbId, min:0,max:1048575,mag:'<%=rb.getString("FanWei")%>：0~1048575'}],
				},
				enbIdTacrowData: [],
				menus_enbIdtac: [],
				enbIdTip: 'false',
				enbIdTacDisabled: false,
				operTacType: 'add',
				AddModifyTacTitle: 'add'
			}
		},
		computed: {
			tableSelectShow() {
				return writableMap.CODE_ADVANCE_SELFSTART == true;
			}
		},
		watch:{
			"idRangeStart":function(val){
				this.modifyMapForm.idRange = this.idRangeStart + "-" + this.idRangeEnd
			},
			"idRangeEnd":function(val){
				this.modifyMapForm.idRange = this.idRangeStart + "-" + this.idRangeEnd
			},
		},
		methods:{
			init(){
				var vm = this;
				axios.post("${ctx}/SON/SelfConfiguration/getSelfConfigurationInfo.action").then(function(response){
					var data = response.data;
					vm.selfForm.selfStartEnable = data.selfStartEnable == "1"?true:false;
					if(data.selfStartEnable == "1"){
						vm.openStartFlag = true;
						vm.showOpenStart = false;
						vm.openStartClass = "disabled";
					}else{
						vm.openStartFlag = false;
						vm.showOpenStart = true;
						vm.openStartClass = "";
					}
					if(writableMap["CODE_ADVANCE_SELFSTART"]){
						
					}else{
						vm.openAllFlag = true;
						vm.openStartFlag = true;
						vm.showOpenStart = false;
					}
					vm.selfForm.executeType = data.executeType;
					vm.selfForm.upgradeEnable = data.upgradeEnable == "1"?true:false;
					vm.selfForm.licenseEnable = data.licenseEnable == "1"?true:false;
					vm.selfForm.selfConfigEnable = data.selfConfigEnable == "1"?true:false;
					vm.selfForm.selfAutoMapEnable = data.selfAutoMapEnable == "1"?true:false;
					//注意这里的回显
					//vm.selfForm.autoConfigMode = data.autoConfigMode == 'MAP'?'0':'1';
					vm.selfForm.defaultTac = data.defaultTac;
					if(data.autoConfigMode == 'MAP'){
						vm.selfForm.autoConfigMode = '0';
					}else if(data.autoConfigMode == 'TAC'){
						vm.selfForm.autoConfigMode = '1';
					}else{
						vm.selfForm.autoConfigMode = '0';
						axios.post("${ctx}/SON/SelfConfiguration/updateSelfConfigurationInfo.action",stringify({'autoConfigMode': 'MAP'})).then(function(response){
							var data = response.data;
							if(data["success"]){
								
							}else{
								//vm.$message.error(data["message"])
							}
						})
					}
					
				})
			},
			changeSwitch(val,type){
				var typeList = {
						"main" : "selfStartEnable",
						"exetype" : "executeType",
						"upgrade" : "upgradeEnable",
						"license" : "licenseEnable",
						"config" : "selfConfigEnable",
						"automap" : "selfAutoMapEnable",
						
						"autoTac" : "autoConfigMode",
						"checkTac" : "defaultTac"
				}
				var params = {},value,vm = this;
				if(type != 'exetype'){
					value = val == true?"1":"0";
				}else{
					value = val;
				}
				if(type == 'autoTac'){
					value = val == '0'?"MAP":"TAC";
				}
				params[typeList[type]] = value;
				axios.post("${ctx}/SON/SelfConfiguration/updateSelfConfigurationInfo.action",stringify(params)).then(function(response){
					var data = response.data;
					if(data["success"]){
						
					}else{
						vm.$message.error(data["message"])
					}
				})
				if(type == 'main' && val){
					vm.openStartFlag = true;
					vm.showOpenStart = false;
					vm.openStartClass = "disabled";
				}else{
					vm.openStartFlag = false;
					vm.showOpenStart = true;
					vm.openStartClass = "";
				}
			},
			handerClose(){
				this.$refs.menu_exe.hide();
			},
			queryExe(){
				Object.assign(this.params_exe,this.params_exe_form);
			},
			queryPlan(){
				Object.assign(this.params_plan,this.params_plan_form);
			},
			queryLicense(){
				Object.assign(this.params_license,this.params_license_form);
			},
			queryUpgrade(){
				Object.assign(this.params_upgrade,this.params_upgrade_form);
			},
			optClick(row,ev){
				var vm = this;
				vm.rowDataExe = row;
				var exeShowFlag;
				var exeEnableFlag;
				var reExeEnableFlag;
				var deleteFlag;
				if(row.execute_type == 1){
					exeShowFlag = true;
				}else{
					exeShowFlag = false;
				}
				if(row.status == 1){
					exeEnableFlag = true;
					reExeEnableFlag = true;
					deleteFlag = true;
				}else if(row.status == 3){
					exeEnableFlag = false;
					reExeEnableFlag = true;
					deleteFlag = false;
				}else if(row.status == 2){
					exeEnableFlag = true;
					reExeEnableFlag = false;
					deleteFlag = false;
				}
				vm.menus_exe = [
					{label:"<%=rb.getString("JieGuo")%>",cls:"el-icon el-icon-operation-result",code:"result"},
					{label:"<%=rb.getString("ChongXinZhiXing")%>",cls:"el-icon el-icon-operation-restart CODE_ADVANCE_SELFSTART hidden",code:"retry",disable:reExeEnableFlag},
					{label:"<%=rb.getString("ZhiXingNew")%>",cls:"el-icon el-icon-operation-start CODE_ADVANCE_SELFSTART hidden",code:"start",disable:exeEnableFlag,show:exeShowFlag},
					{label:"<%=rb.getString("ShanChu")%>",cls:"el-icon el-icon-operation-delete CODE_ADVANCE_SELFSTART hidden",code:"del",disable:deleteFlag}
				]
				vm.$nextTick(function(){
					document.body.click();
					vm.$refs.menu_exe.show(ev)
				})
			},
			clickMenu(ev){
				var codes = {
						result:this.viewConfigTask,
						start:this.exeConfigTask,
						retry:this.reExeConfigTask,
						del:this.deleteConfigTask
				}
				if(codes[ev.code]){
					codes[ev.code]()
				}
			},
			viewConfigTask(){
				var vm = this;
				vm.slideUrl = '${ctx}/SON/SelfConfigurationTask/goSelfTaskProgress.action?rd='+Math.random(),
				vm.slideTitle = '<%=rb.getString("JieGuo")%>';
				vm.slideFooter = false;
				vm.slideHeader = true;
				vm.slidePosition = 'bottom';
				vm.slideHeight = '350px';
				vm.slideWidth = '100%';
				vm.configType = "exe";
				vm.$refs.slide.showSlide(function(){
	    	    	vm.slideModal = false;
	    	    });
			},
			exeConfigTask(){
				var vm = this;
				var params = {
						taskId : vm.rowDataExe.task_id,
						serialNumber : vm.rowDataExe.serial_number
				}
				axios.post("${ctx}/SON/SelfConfigurationTask/exeNextSelfConfigTask.action",stringify(params)).then(function(response){
					var data = response.data;
					if(data["success"]){
						vm.$refs.ctableExe.refresh();
					}else{
						vm.$message.error(data["message"]);
					}
				})
			},
			reExeConfigTask(){
				var vm = this;
				var params = {
						taskId : vm.rowDataExe.task_id,
						serialNumber : vm.rowDataExe.serial_number,
						execute_type : vm.rowDataExe.execute_type
				}
				axios.post("${ctx}/SON/SelfConfigurationTask/reExeSelfConfigTask.action",stringify(params)).then(function(response){
					var data = response.data;
					if(data["success"]){
						vm.$refs.ctableExe.refresh();
					}else{
						vm.$message.error(data["message"]);
					}
				})
			},
			deleteConfigTask(){
				var vm = this;
				var confirmStr = '<%=rb.getString("QueRenShanChu")%>';
				var params = {
						taskId : vm.rowDataExe.task_id
				}
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					axios.post("${ctx}/SON/SelfConfigurationTask/delSelfConfigTask.action",stringify(params)).then(function(response){
						var data = response.data;
						var message = '<%=rb.getString("ChengGong")%>';
						if(data["success"]){
							vm.$message({
	    						message:message,
	    						type:'success',
	    					})
                            vm.$refs.ctableExe.refresh();
						}else{
							vm.$message.error(data["message"])
						}
					})
				}).catch(() => {
					
				})
			},
			handerCloseUpgrade(){
				this.$refs.menu_upgrade.hide();
			},
			optClickUpgrade(row,ev){
				var vm = this;
				if(vm.openStartClass == "disabled"){
					
				}else{
					vm.rowDataUpgrade = row;
					var text,icon;
					if(row.status == "0"){
						text = "<%=rb.getString("QiYong")%>";
						icon = "el-icon el-icon-operation-enable1";
					}else{
						text = "<%=rb.getString("JinYong")%>";
						icon = "el-icon el-icon-operation-disable1"
					}
					vm.menus_upgrade = [
						{label:"<%=rb.getString("XiuGai")%>",cls:"el-icon el-icon-operation-edit",code:"edit"},
						{label:text,cls:icon,code:"enable"},
						{label:"<%=rb.getString("ShanChu")%>",cls:"el-icon el-icon-operation-delete",code:"del"}
					]
					vm.$nextTick(function(){
						document.body.click();
						vm.$refs.menu_upgrade.show(ev);
					})
				}
			},
			clickMenuUpgrade(ev){
				var codes = {
						edit:this.editUpgrade,
						enable:this.enableUpgrade,
						del:this.delUpgrade
				}
				if(codes[ev.code]){
					codes[ev.code]()
				}
			},
			handerClosePlan(){
				this.$refs.menu_configPlan.hide();
			},
			optClickPlan(row,ev){
				var vm = this;
				if(vm.openStartClass == "disabled"){
					
				}else{
					vm.rowDataPlan = row;
					vm.menus_configPlan = [
						{label:"<%=rb.getString("XinXi")%>",cls:"el-icon el-icon-operation-info",code:"info"},
						{label:"<%=rb.getString("XiuGai")%>",cls:"el-icon el-icon-operation-edit CODE_ADVANCE_SELFSTART hidden",code:"edit",platform: row.platform},
						{label:"<%=rb.getString("ShanChu")%>",cls:"el-icon el-icon-operation-delete CODE_ADVANCE_SELFSTART hidden",code:"del"}
					]
					vm.$nextTick(function(){
						document.body.click();
						vm.$refs.menu_configPlan.show(ev)
					})
				}
			},
			clickMenuPlan(ev){
				var codes = {
						info:this.viewConfigPlan,
						edit:this.editConfigPlan,
						del:this.delConfigPlan
				}
				if(codes[ev.code]){
					codes[ev.code](ev)
				}
			},
			handerCloseRule(){
				this.$refs.menu_configRule.hide();
			},
			optClickRule(row,ev){
				var vm = this;
				if(vm.openStartClass == "disabled"){
					
				}else{
					vm.rowDataRule = row;
					var text,icon;
					if(row.switch_enable == "0"){
						text = "<%=rb.getString("QiYong")%>";
						icon = "el-icon el-icon-operation-enable1 CODE_ADVANCE_SELFSTART hidden";
					}else{
						text = "<%=rb.getString("JinYong")%>";
						icon = "el-icon el-icon-operation-disable1 CODE_ADVANCE_SELFSTART hidden"
					}
					vm.menus_configRule = [
						{label:"<%=rb.getString("XinXi")%>",cls:"el-icon el-icon-operation-info",code:"info"},
						{label:text,cls:icon,code:"enable"},
						{label:"<%=rb.getString("XiuGai")%>",cls:"el-icon el-icon-operation-edit CODE_ADVANCE_SELFSTART hidden",code:"edit",platform: row.platform},
						{label:"<%=rb.getString("XiaZai")%>",cls:"el-icon el-icon-operation-download",code:"download"}
					]
					vm.$nextTick(function(){
						document.body.click();
						vm.$refs.menu_configRule.show(ev)
					})
				}
				
			},
			clickMenuRule(ev){
				var codes = {
						info:this.viewConfigRule,
						enable:this.enableConfigRule,
						edit:this.editConfigRule,
						download:this.downloadConfigRule
				}
				if(codes[ev.code]){
					codes[ev.code](ev);
				}
			},
			addUpgrade(){
				var vm = this;
				vm.slideUrl = '${ctx}/SON/SelfConfiguration/goSelfUpgradeOper.action',
				vm.slideTitle = '<%=rb.getString("ShengJiGuiZeChuangJian")%>';
				vm.slideFooter = true;
				vm.slideHeader = true;
				vm.slidePosition = 'top';
				vm.slideHeight = '100%';
				vm.slideWidth = '100%';
				vm.configType = 'upgrade';
				vm.operTypeUpgrade = "add";
				vm.$refs.slide.showSlide(function(){
	    	    	vm.slideModal = false
	    	    });
			},
			editUpgrade(){
				var vm = this;
				vm.slideUrl = '${ctx}/SON/SelfConfiguration/goSelfUpgradeOper.action',
				vm.slideTitle = '<%=rb.getString("ShengJiGuiZeXiuGai")%>';
				vm.slideFooter = true;
				vm.slideHeader = true;
				vm.slidePosition = 'top';
				vm.slideHeight = '100%';
				vm.slideWidth = '100%';
				vm.configType = 'upgrade';
				vm.operTypeUpgrade = "edit";
				vm.$refs.slide.showSlide(function(){
	    	    	vm.slideModal = false;
	    	    	eventBus.$emit("edit-upgrade");
	    	    });
			},
			enableUpgrade(){
				var vm = this;
				var params = {
						id : vm.rowDataUpgrade.id
				}
				if(vm.rowDataUpgrade.status == "0"){
					params.state = "1"
				}else{
					params.state = "0"
				}
				axios.post("${ctx}/SON/SelfConfiguration/changeSelfUpgradePlansState.action",stringify(params)).then(function(response){
					var data = response.data;
					if(data["success"]){
						vm.$refs.ctableUpgrade.refresh();
					}else{
						vm.$message.error(data["message"]);
					}
				})
			},
			delUpgrade(){
				var vm = this;
				var confirmStr = '<%=rb.getString("QueRenShanChu")%>';
				var params = {
						id:vm.rowDataUpgrade.id
				}
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					axios.post("${ctx}/SON/SelfConfiguration/delSelfUpgradePlans.action",stringify(params)).then(function(response){
						var data = response.data;
						var message = '<%=rb.getString("ChengGong")%>';
						if(data["success"]){
							vm.$message({
	    						message:message,
	    						type:'success',
	    					})
                            vm.$refs.ctableUpgrade.refresh();
						}else{
							vm.$message.error(data["message"])
						}
					})
				}).catch(() => {
					
				})
			},
			cancelSlide(){
				if(this.configType == "plan"){
					if(this.operTypePlan == "view"){
						this.$refs.slide.hide();
					}else{
						eventBus.$emit("cancel-plan");
						eventBus.$emit("cancel-rule");
					}
				}
				if(this.configType == "rule"){
					if(this.operTypeRule == "view"){
						this.$refs.slide.hide();
					}else{
						eventBus.$emit("cancel-rule");
					}
				}
				if(this.configType == "upgrade" || this.configType == "exe"){
					this.$refs.slide.hide();
				}
			},
			saveSlide(){
				if(this.configType == "plan"){
					eventBus.$emit("save-plan");
				}
				if(this.configType == "rule"){
					eventBus.$emit("save-rule");
				}
				if(this.configType == "upgrade"){
					eventBus.$emit("save-upgrade");
				}
			},
			viewLicense(){
				this.licenseVisible = true;
			},
			showExportType(){
				this.exportTypeFlag = !this.exportTypeFlag
			},
			handerCloseExport(){
				this.exportTypeFlag = false;
			},
			viewConfigPlan(){
				var vm = this;
				vm.slideUrl = '${ctx}/SON/SelfConfiguration/goSelfParamConfigEdit.action',
				vm.slideTitle = '<%=rb.getString("ChaKan")%> <%=rb.getString("PeiZhiGuiHua")%>';
				vm.slideFooter = false;
				vm.slideHeader = true;
				vm.slidePosition = 'top';
				vm.slideHeight = '100%';
				vm.slideWidth = '100%';
				vm.operTypePlan = "view";
				vm.configType = "plan";
				vm.$refs.slide.showSlide(function(){
	    	    	vm.slideModal = false
	    	    });
			},
			editConfigPlan(row){
				var vm = this;
				
				if(['CR-B4860','QRTB','MLN'].includes(row.platform)) {
					vm.slideUrl = '${ctx}/SON/SelfConfiguration/goMultiCarrierRuleConfigPage.action?platformType='+row.platform;
				}else {
					vm.slideUrl = '${ctx}/SON/SelfConfiguration/goSelfParamConfigEdit.action';
				}
				sessionStorage.setItem('paramType','special');
				
				vm.slideTitle = '<%=rb.getString("XiuGai")%> <%=rb.getString("PeiZhiGuiHua")%>';
				vm.slideFooter = true;
				vm.slideHeader = true;
				vm.slidePosition = 'top';
				vm.slideHeight = '100%';
				vm.slideWidth = '100%';
				vm.operTypePlan = "edit";
				vm.configType = "plan";
				vm.$refs.slide.showSlide(function(){
	    	    	vm.slideModal = false;
	    	    });
			},
			delConfigPlan(){
				var vm = this;
				var confirmStr = '<%=rb.getString("QueRenShanChu")%>';
				var params = {
						id:vm.rowDataPlan.id,
						platform:vm.rowDataPlan.platform
				}
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					axios.post("${ctx}/SON/SelfConfiguration/delSelfConfigPlans.action",stringify(params)).then(function(response){
						var data = response.data;
						var message = '<%=rb.getString("ChengGong")%>';
						if(data["success"]){
							vm.$message({
	    						message:message,
	    						type:'success',
	    					})
                            vm.$refs.ctablePlan.refresh();
						}else{
							vm.$message.error(data["message"])
						}
					})
				}).catch(() => {
					
				})
			},
			productFmt(row,column,value,index){
				/* 产品类型fmt */
				if(value == 1){
					return "RTS";
				}else if(value == 2){
					return "QAFA / QAFB";
				}else if(value == 3){
					return "HaloB";
				}else if(value == 4){
					return "RTD"
				}
			},
			viewConfigRule(){
				var vm = this;
				vm.slideUrl = '${ctx}/SON/SelfConfiguration/goSelfParamConfigEditRule.action',
				vm.slideTitle = '<%=rb.getString("ChaKan")%> <%=rb.getString("PeiZhiGuiZe")%>';
				vm.slideFooter = false;
				vm.slideHeader = true;
				vm.slidePosition = 'top';
				vm.slideHeight = '100%';
				vm.slideWidth = '60%';
				vm.operTypeRule = 'view';
				vm.configType = 'rule';
				vm.$refs.slide.showSlide(function(){
	    	    	vm.slideModal = false
	    	    });
			},
			enableConfigRule(){
				var vm = this;
				var params = {
						platform : vm.rowDataRule.platform
				}
				if(vm.rowDataRule.switch_enable == "0"){
					params.switchEnable = "1"
				}else{
					params.switchEnable = "0"
				}
				axios.post("${ctx}/SON/SelfConfiguration/updateSelfConfigRuleSwitchEnable.action",stringify(params)).then(function(response){
					var data = response.data;
					if(data["success"]){
						vm.$refs.ctableRule.refresh();
					}else{
						vm.$message.error(data["message"]);
					}
				})
			},
			editConfigRule(row){
				var vm = this;
				
				if(['CR-B4860','QRTB','MLN'].includes(row.platform)) {
					vm.slideUrl = '${ctx}/SON/SelfConfiguration/goMultiCarrierRuleConfigPage.action?platformType='+row.platform;
				}else {
					vm.slideUrl = '${ctx}/SON/SelfConfiguration/goSelfParamConfigEditRule.action';
				}
				sessionStorage.setItem('paramType','auto');
				
				vm.slideTitle = '<%=rb.getString("XiuGai")%> <%=rb.getString("PeiZhiGuiZe")%>';
				vm.slideFooter = true;
				vm.slideHeader = true;
				vm.slidePosition = 'top';
				vm.slideHeight = '100%';
				vm.slideWidth = '100%';
				vm.operTypeRule = 'edit';
				vm.configType = 'rule';
				vm.$refs.slide.showSlide(function(){
	    	    	vm.slideModal = false
	    	    });
			},
			downloadConfigRule(){
				var params = {
						platform:this.rowDataRule.platform
				}
				var url = "${ctx}/SON/SelfConfiguration/doDownloadConfigParamFile.action";
				exportByForm(url,params);
			},
			bandWidthFmt(row,column,value,index){
				if(value == "n25"){
					return "5";
				}else if(value == "n50"){
					return "10"
				}else if(value == "n75"){
					return "15"
				}else if(value == "n100"){
					return "20"
				}
			},
			earfcnFmt(row,column,value,index){
				if(value == "" || value == null){
					return "";
				}else{
					return translateToFre(value);
				}
			},
			sfassignmentFmt(row,column,value,index){
				if(value == "1"){
					return "1(DL:UL = 2:2)"
				}else if(value =="2"){
					return "2(DL:UL = 3:1)"
				}else if(value == "0"){
					return "0(DL:UL = 1:3)"
				}else if(value == "6"){
					return "6(DL:UL = 3:5)"
				}else{
					return value
				}
			},
			importConfigPlan(){
				var vm = this;
				vm.importVisible = true;
			},
			cancelImportPlan(){
				this.importVisible = false;
				this.importForm = {
					importType:"0",
					filePath:""
				}
				$("#uploadForm_configPlan input[name='uploadFile']").val("")
				this.$refs.importForm.resetFields();
			},
			importFile(){
				$("#uploadForm_configPlan input[name='uploadFile']").click();
			},
			exportTemp(id){
				var params = {
						tempType:id
				}
				var url = "${ctx}/SON/SelfConfiguration/downloadSelfConfigTemp.action";
				exportByForm(url,params);
			},
			exportConfigPlan(){
				var params = {
						search_text : this.params_plan.searchText,
						timeZone : timeZone
				}
				var url = "${ctx}/SON/SelfConfiguration/downloadSelfConfigPlanningResList.action"
				exportByForm(url,params);
			},
			executeStatus(row,column,value,rowIndex){
				if ('0' == value) {
					return "<%=rb.getString("WeiZhiXing")%>";
				} else if ('1' == value){
					return "<%=rb.getString("ZhengZaiZhiXing")%>";
				} else if ('2' == value){
					return "<%=rb.getString("ZhiXingChengGong")%>";
				} else if ('3' == value){
					return "<%=rb.getString("ZhiXingShiBai")%>";
				}
			},
			retainConfigFmt(row,column,value,index){
				if(value == "0"){
					return "<%=rb.getString("Fou")%>";
				}else if(value == "1"){
					return "<%=rb.getString("Shi")%>";
				}
			},
			saveImportFile(){
				var vm = this;
				vm.$refs.importForm.validate((valid) => {
					if(valid){
						vm.importLoading = true;
						var files = document.querySelector("#uploadFileConfigPlan").files;
						$("#uploadForm_configPlan [name=fileSize]").val(files[0].size);
				    	$("#uploadForm_configPlan [name=operType]").val(vm.importForm.importType);
						<%-- $("#uploadForm_configPlan").form('submit', {
						     url: "${ctx}/SON/SelfConfiguration/importSelfConfigPlanningInfos.action",
						     success: function (data) {
						    	 vm.importLoading = false;
						    	var data = JSON.parse(data);
						     	if(data["success"]){
						     		vm.cancelImportPlan();
						     		$("#uploadForm_configPlan input[name='uploadFile']").val("");
						     		vm.$message({
			    						message:"<%=rb.getString("ChengGong")%>",
			    						type:'success'
			    					})
						     		vm.$refs.ctablePlan.refresh();
						     	}else{
						     		if(data["msg"] == "1"){
						     			vm.$message.error('<%=rb.getString("DaoRuShiBai")%>')
						     		}else if(data["msg"] == "2"){
						     			vm.cancelImportPlan();
						     			$("#uploadForm_configPlan input[name='uploadFile']").val("");
						     			vm.$refs.ctablePlan.refresh();
						     			vm.downloadVisible = true;
					        		}else{
					        			vm.$message.error('<%=rb.getString("DaoRuShiBai")%>')
					        		}
						     	}
						     }
						}) --%>
						uploadWithProgress({
							url: "${ctx}/SON/SelfConfiguration/importSelfConfigPlanningInfos.action",
							form: document.querySelector("#uploadForm_configPlan"),
							success: function (data) {
						    	vm.importLoading = false;
						     	if(data["success"]){
						     		vm.cancelImportPlan();
						     		$("#uploadForm_configPlan input[name='uploadFile']").val("");
						     		vm.$message({
			    						message:"<%=rb.getString("ChengGong")%>",
			    						type:'success'
			    					})
						     		vm.$refs.ctablePlan.refresh();
						     	}else{
						     		if(data["msg"] == "1"){
						     			vm.$message.error('<%=rb.getString("DaoRuShiBai")%>')
						     		}else if(data["msg"] == "2"){
						     			vm.cancelImportPlan();
						     			$("#uploadForm_configPlan input[name='uploadFile']").val("");
						     			vm.$refs.ctablePlan.refresh();
						     			vm.downloadVisible = true;
					        		}else{
					        			vm.$message.error('<%=rb.getString("DaoRuShiBai")%>')
					        		}
						     	}
						     }
						});
					}
				})
			},
			exeTypeConfigFmt(row,column,value,index){
				if(value == "0"){
					return "<%=rb.getString("ZiDongZhiXing")%>"
				}else if(value == "1"){
					return "<%=rb.getString("ShouDongZhiXing")%>"
				}
			},
			statusConfigFmt(row,column,value,index){
				if(value == "0"){
					return "<%=rb.getString("DengDai")%>"
				}else if(value == "1"){
					return "<%=rb.getString("JinXingZhong")%>"
				}else if(value == "2"){
					return "<%=rb.getString("YiJieShu")%>"
				}else if(value == "3"){
					return "<%=rb.getString("DaiZhiXing")%>"
				}
			},
			resultConfigFmt(row,column,value,index){
				if(value == "0"){
					return "<%=rb.getString("ChengGong")%>"
				}else if(value == "1"){
					return "<%=rb.getString("ShiBai")%>"
				}else if(value == "2"){
					return "<%=rb.getString("BuFenChengGong")%>"
				}
			},
			clickTab(){
				this.$refs.slide.hide();
			},
			downloadErrorFile(){
				var url = "${ctx}/SON/SelfConfiguration/downloadFailureFile.action"
				exportByForm(url,{});
				this.downloadVisible = false;
			},
			selectPolFile(){
				this.$refs['file_polygon'].click();
			},
			beforeUploadPolygon(file){
				var fd = new FormData(),vm = this,
				config = {
					headers: { 'Content-Type': 'multipart/form-data' }
				};
				fd.append('uploadFile',file);
				 //文件流
				axios.post("${ctx}/SON/SelfConfiguration/importSelfCoordinateTac.action",fd,config).then(function(response){
					var data = response.data
					if(data["success"]){
			     		vm.$refs.ctableMap.refresh();
			     		vm.closeMapImport();
					}else{
						vm.$message.error(data["message"])
					}
					vm.mapLoading = false;
				})
			},
			fileChangePolygon(file,fileList){
				var vm = this;
				vm.mapForm.file_polygon = file.name;
			},
			
			//tac auto configure: eNB ID - TAC
			addMapTac(){
				var vm = this;
				vm.addTacVisible = true;
				vm.operTacType = 'add';
				vm.AddModifyTacTitle = '<%=rb.getString("TianJia")%>';
				vm.enbIdTacDisabled = false;
			},	
			enbIdTacOptClick(row,ev){
				var vm = this;
				vm.enbIdTacrowData = row;
				vm.menus_enbIdtac = [
					{label:'<%=rb.getString("XiuGai")%>',cls:'el-icon el-icon-operation-edit',code:'edit'},
					{label:'<%=rb.getString("ShanChu")%>',cls:'el-icon el-icon-operation-delete',code:'del'}
				]
				vm.$nextTick(function(){
					document.body.click();
					vm.$refs.menu_enbIdtac.show(ev)
				})
			},
			enbIdTacClickMenu(ev){
				var vm = this;
				var codeList = {
						edit : vm.enbIdTacEdit,
						del : vm.enbIdTacDelete
				}
				codeList[ev.code]();
			},
			enbIdTachanderClose(){
				this.$refs.menu_enbIdtac.hide();
			},
			enbIdTacEdit(){
				var vm = this;
				vm.addTacVisible = true;
				vm.operTacType = 'update';
				vm.AddModifyTacTitle = '<%=rb.getString("XiuGai")%>';
				vm.enbIdTacDisabled = true;
				vm.addTacForm.tac = vm.enbIdTacrowData.tac;
				vm.addTacForm.eciRange = vm.enbIdTacrowData.eciRange;
			},								
			addTacSubmit(){
				var vm = this;
				//修改时校验参数是否发生变化
				vm.$refs.addTacForm.validate(function(valid){
					if(valid){
						var params = {
								tac : vm.addTacForm.tac,
								eciRange : vm.addTacForm.eciRange
						}
						if(vm.operTacType == 'add'){
							params.operType = 'add';
						}else{
							params.operType = 'update';
						}
						axios.post("${ctx}/SON/SelfConfiguration/setSelfAutoTac.action",stringify(params)).then(function(response){
							var data = response.data;
							var message = '<%=rb.getString("ChengGong")%>';
							if(data["success"]){
								vm.$message({
		    						message:message,
		    						type:'success'
		    					})
		    					vm.$refs.enbIdTactable.refresh();
								vm.addTacCancel();
							}else{
								vm.$message.error(data["message"])
							}
						})
					}
				})
			},			
			addTacCancel(){
				var vm = this;
				vm.addTacVisible = false;
				vm.$refs.addTacForm.resetFields();
			},
			enbIdTacDelete(){
				var vm = this;
				var confirmStr = '<%=rb.getString("QueRenShanChu")%>';
				var params = {
						id: vm.enbIdTacrowData.id
				}
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					axios.post("${ctx}/SON/SelfConfiguration/delSelfAutoTac.action",stringify(params)).then(function(response){
						var data = response.data;
						var message = '<%=rb.getString("ChengGong")%>';
						if(data["success"]){
							vm.$message({
	    						message:message,
	    						type:'success'
	    					})
	    					vm.$refs.enbIdTactable.refresh();
						}else{
							vm.$message.error(data["message"])
						}
					})
				}).catch()
			},
	
			//tac auto configure: Coordinate - TAC
			importMapFile(){
				this.showMapImport = true;
			},
			clearMapFile(){
				var vm = this;
				var confirmStr = '<%=rb.getString("QueRenShanChu")%>';
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					axios.post("${ctx}/SON/SelfConfiguration/clearSelfCoordinateTac.action").then(function(response){
						var data = response.data;
						var message = '<%=rb.getString("ChengGong")%>';
						if(data["success"]){
							vm.$message({
	    						message:message,
	    						type:'success'
	    					})
	    					vm.$refs.ctableMap.refresh();
						}else{
							vm.$message.error(data["message"])
						}
					})
				}).catch()
			},
			closeMapImport(){
				this.showMapImport = false;
				this.mapForm.file_polygon = '';
				this.$refs.upload_polygon.clearFiles();
				this.$refs.mapForm.resetFields();
			},
			deleteMap(){
				var vm = this;
				var confirmStr = '<%=rb.getString("QueRenShanChu")%>';
				var params = {
						id:vm.rowDataMap.id
				}
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					axios.post("${ctx}/SON/SelfConfiguration/delSelfCoordinateTacById.action",stringify(params)).then(function(response){
						var data = response.data;
						var message = '<%=rb.getString("ChengGong")%>';
						if(data["success"]){
							vm.$message({
	    						message:message,
	    						type:'success'
	    					})
	    					vm.$refs.ctableMap.refresh();
						}else{
							vm.$message.error(data["message"])
						}
					})
				}).catch()
			},
			submitMap(){
				var vm = this;
				vm.$refs.mapForm.validate((valid) => {
					if(valid){
						vm.$refs.upload_polygon.submit();
						vm.mapLoading = true;
					}
				})
			},
			optClickMap(row,ev){
				var vm = this;
				vm.rowDataMap = row;
				vm.menus_map = [
					{label:'<%=rb.getString("XiuGai")%>',cls:'el-icon el-icon-operation-edit',code:'edit'},
					{label:'<%=rb.getString("ShanChu")%>',cls:'el-icon el-icon-operation-delete',code:'del'}
				]
				vm.$nextTick(function(){
					document.body.click();
					vm.$refs.menu_map.show(ev)
				})
			},
			clickMenuMap(ev){
				var vm = this;
				var codeList = {
						edit : vm.editMap,
						del : vm.deleteMap
				}
				codeList[ev.code]();
			},
			editMap(){
				this.modifyMapVisible = true;
				this.modifyMapForm.groupName = this.rowDataMap.groupName;
				this.modifyMapForm.tac = this.rowDataMap.tac;
				this.idRangeStart = (this.rowDataMap.eciRange).split("-")[0];
				this.idRangeEnd = (this.rowDataMap.eciRange).split("-")[1];
			},
			handerCloseMap(){
				this.$refs.menu_map.hide();
			},
			editMapSubmit(){
				var vm = this;
				vm.$refs.modifyMapForm.validate(function(valid){
					if(valid){
						var params = {
								id : vm.rowDataMap.id,
								tac : vm.modifyMapForm.tac,
								groupName :vm.modifyMapForm.groupName,
								eciRange:vm.modifyMapForm.idRange
						}
						axios.post("${ctx}/SON/SelfConfiguration/updateSelfCoordinateTacById.action",stringify(params)).then(function(response){
							var data = response.data;
							var message = '<%=rb.getString("ChengGong")%>';
							if(data["success"]){
								vm.$message({
		    						message:message,
		    						type:'success'
		    					})
		    					vm.$refs.ctableMap.refresh();
								vm.modifyMapVisible = false;
							}else{
								vm.$message.error(data["message"])
							}
						})
					}
				})
			},
			cancelEditMap(){
				this.modifyMapVisible = false;
			}
		},
		mounted(){
			var vm = this;
			vm.init();
			$("#uploadForm_configPlan input[name='uploadFile']").bind("change", function() {
				vm.importForm.filePath = this.value
			});
		}
	})
</script>