<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<link rel="stylesheet" href="${ctx}/js/topo/dvf.css" type="text/css" media="screen"/>
<link rel="stylesheet" href="${ctx}/js/topo/node.css" type="text/css" media="screen"/>
<link rel="stylesheet" href="${ctx}/js/packages/topo/enb-map.css" type="text/css" media="screen"/>
<div class="overflow-cls">
	<div id="topo_container" style="height: 99.5%;width: 100%;min-width: 900px;overflow: hidden;display: flex;padding: 2px 0px 0px 5px;background: #fff;">
		<!-- Device group -->
		<div style="width: 360px;height: 100%;position: relative;" class="flex-ctn" v-show="groupShow">
			<div class="flex-ctn absolute-ctn">
				<div v-if="activeType == 'site'" style="position: absolute;z-index: 100;left: 310px;top: 6px;">
					<span @click="addSite" class="el-icon el-icon-operation-add"></span>
				</div>
				<el-tabs style="height: 100%;position: relative;" v-model="activeType" @tab-click="tabClick">
					<el-tab-pane label="<%=rb.getString("SheBeiZu")%>" name="group">
						<div style="flex: auto;height: 100%;" @click="observSlider">
							<el-ctable ref="group" :pagination="false" :rownumber="false" :query-params="groupQueryParams" :url="groupURL" @selection-change="groupChange" row-key="id"
								@load-success="loadSuccess">
								<template slot="toolbar">
									<el-query style="zoom: 0.8;" class="width-fitcontent" type="normal" @query="queryGroup" placeholder='<%=rb.getString("SheBeiZuMingCheng")%>/<%=rb.getString("HostName")%>'></el-query>
								</template>
								<el-table-column type="selection"></el-table-column>
								<el-table-column label="<%=rb.getString("SheBeiZuMingCheng")%>" prop="group_name" :show-overflow-tooltip="false">
									<template slot-scope="scope">
										<div style="display: flex;align-items: center;justify-content:space-between;">
											<span class="ellipsis-txt" style="width: 260px;" :title="scope.row.group_name">{{scope.row.group_name}}</span>
											<i class="row-op el-icon el-icon-arrow-right" @click="rowClick(scope.row)"></i>
										</div>
									</template>
								</el-table-column>
							</el-ctable>
						</div>
					</el-tab-pane>
					<el-tab-pane v-if="siteEnable" label='<%=rb.getString("ZhanDianMingCheng")%>' name="site">
						<div style="flex: auto;overflow: auto;height: 100%;">
							<el-ctable ref="siteList" :rownumber="false" row-key="id" height="100%"
								row-key="siteName"
								:url="siteListURL"
								:query-params="siteQuery"
								:front-pagination="true"
								:pagination="true"
								@row-dblclick="rowdblclickSite"
								@row-click="siteCurrentChange"
								@load-success="getSiteList"
							>
								<template slot="toolbar">
									<el-query style="zoom: 0.8;" class="width-fitcontent" type="normal" @query="querySite" placeholder='<%=rb.getString("ZhanZhiMingCheng")%>'></el-query>
								</template>
								<el-table-column label='' prop="status" width="40">
									<template slot-scope="scope">
										<div v-if="scope.row.status == '1'" class='el-icon el-icon-status-conn-on' style='font-size:20px;'></div>
										<div v-if="scope.row.status == '2'" class='el-icon el-icon-status-conn-on abnormal-color' style='font-size:20px;'></div>
										<div v-if="scope.row.status == '3'" class='el-icon el-icon-status-conn-off' style='font-size:20px;'></div>
									</template>
								</el-table-column>
								<el-table-column label='<%=rb.getString("ZhanZhiMingCheng")%>' prop="siteName"></el-table-column>
								<el-table-column label='<%=rb.getString("WeiDu")%>' width="75" prop="latitude"></el-table-column>
								<el-table-column label='<%=rb.getString("JingDu")%>' width="82" prop="longitude"></el-table-column>
								<el-table-column width="70">
									<template slot-scope="scope">
										<span @click="modifySite(scope.row, event)" class="el-icon el-icon-operation-edit" style="margin-right: 5px;"></span>
										<span @click="delSite(scope.row, event)" class="el-icon el-icon-operation-delete"></span>
									</template>
								</el-table-column>
							</el-ctable>
						</div>
					</el-tab-pane>
				</el-tabs>
			</div>
			<!-- device slider -->
			<div ref="deviceSlider" class="device-slider flex-ctn">
				<!-- device list -->
				<div style="flex: auto;overflow: auto;">
					<el-tabs style="height: 100%;" class="is-bottom-active">
						<el-tab-pane label="<%=rb.getString("XiaoZhan")%>">
							<el-ctable ref="enbDevice" @row-dblclick="rowdblclick" data-key="ENB" :rownumber="true" :url="deviceURL" :query-params="enbQueryParams" :front-pagination="true">
								<template slot="toolbar">
									<div style="display: flex; flex-wrap: wrap; align-items: center;padding: 0 10px;">
										<el-query style="zoom: 0.8;" ref="enblist" class="no-margin" type="normal" @query="queryDevice" placeholder="<%=rb.getString("SheBeiMingCheng")%>/<%=rb.getString("HostName")%>"></el-query>
										
										<div style="padding: 5px 0 0 0;zoom: 0.9;">
											<el-checkbox v-model="enbQueryParams.no_gps" true-label="" false-label="1"><%=rb.getString("WuJingWeiDuSheBei")%></el-checkbox>
											<span v-if="hasSAS" style="color: #a9a9a9;"><%=rb.getString("WuJingWeiDuSheBeiTiShi")%></span>
										</div>
									</div>
								</template>
								<el-table-column label="<%=rb.getString("XiaoZhanBianMa")%>" prop="serial_number" min-width="120"></el-table-column>
								<el-table-column label="<%=rb.getString("HostName")%>" prop="host_name"></el-table-column>
								<el-table-column label="<%=rb.getString("WeiDu")%>" prop="latitude"></el-table-column>
								<el-table-column label="<%=rb.getString("JingDu")%>" prop="longitude"></el-table-column>
								<el-table-column width="45">
									<template slot-scope="scope">
										<div v-if="!sasEnable" class="el-icon el-icon-operation-settings" @click="optClick(scope.row,event)"></div>
										<div v-if="sasEnable" class="el-icon el-icon-operation-settings disabled"></div>
									</template>
								</el-table-column>
							</el-ctable>
						</el-tab-pane>
						<el-tab-pane v-if="gnbEnable" label="<%=rb.getString("gNB")%>">
							<el-ctable ref="gnbDevice" @row-dblclick="rowdblclick" data-key="GNB" :rownumber="true" :url="deviceURL" :query-params="gnbQueryParams" :front-pagination="true">
								<template slot="toolbar">
									<div style="display: flex; flex-wrap: wrap; align-items: center;padding: 0 10px;">
										<el-query style="zoom: 0.8;" ref="enblist" class="no-margin" type="normal" @query="queryGNBDevice" placeholder="<%=rb.getString("SheBeiMingCheng")%>/<%=rb.getString("HostName")%>"></el-query>
										
										<div style="padding: 5px 0 0 0;zoom: 0.9;">
											<el-checkbox v-model="gnbQueryParams.no_gps" true-label="" false-label="1"><%=rb.getString("WuJingWeiDuSheBei")%></el-checkbox>
											<span v-if="hasSAS" style="color: #a9a9a9;"><%=rb.getString("WuJingWeiDuSheBeiTiShi")%></span>
										</div>
									</div>
								</template>
								<el-table-column label="<%=rb.getString("XiaoZhanBianMa")%>" prop="serial_number" min-width="120"></el-table-column>
								<el-table-column label="<%=rb.getString("HostName")%>" prop="host_name"></el-table-column>
								<el-table-column label="<%=rb.getString("WeiDu")%>" prop="latitude"></el-table-column>
								<el-table-column label="<%=rb.getString("JingDu")%>" prop="longitude"></el-table-column>
								<el-table-column width="45">
									<template slot-scope="scope">
										<div v-if="!sasEnable" class="el-icon el-icon-operation-settings" @click="optClick(scope.row,event)"></div>
										<div v-if="sasEnable" class="el-icon el-icon-operation-settings disabled"></div>
									</template>
								</el-table-column>
							</el-ctable>
						</el-tab-pane>
						<el-tab-pane v-if="isGSMEnable" label="GSM">
							<el-ctable ref="gsmDevice" @row-dblclick="rowdblclick" data-key="GSM" :rownumber="true" :url="deviceURL" :query-params="gsmQueryParams" :front-pagination="true">
								<template slot="toolbar">
									<div style="display: flex; flex-wrap: wrap; align-items: center;padding: 0 10px;">
										<el-query style="zoom: 0.8;" ref="enblist" class="no-margin" type="normal" @query="queryGSMDevice" placeholder="<%=rb.getString("SheBeiMingCheng")%>/<%=rb.getString("HostName")%>"></el-query>
										
										<div style="padding: 5px 0 0 0;zoom: 0.9;">
											<el-checkbox v-model="gsmQueryParams.no_gps" true-label="" false-label="1"><%=rb.getString("WuJingWeiDuSheBei")%></el-checkbox>
										</div>
									</div>
								</template>
								<el-table-column label="<%=rb.getString("XiaoZhanBianMa")%>" prop="serial_number" min-width="120"></el-table-column>
								<el-table-column label="<%=rb.getString("HostName")%>" prop="host_name"></el-table-column>
								<el-table-column label="<%=rb.getString("WeiDu")%>" prop="latitude"></el-table-column>
								<el-table-column label="<%=rb.getString("JingDu")%>" prop="longitude"></el-table-column>
								<el-table-column width="45">
									<template slot-scope="scope">
										<div v-if="!sasEnable" class="el-icon el-icon-operation-settings" @click="optClick(scope.row,event)"></div>
										<div v-if="sasEnable" class="el-icon el-icon-operation-settings disabled"></div>
									</template>
								</el-table-column>
							</el-ctable>
						</el-tab-pane>
						<el-tab-pane label="<%=rb.getString("CPE")%>" v-if="writableMap['CODE_CPE_MONITOR'] != undefined">
							<el-ctable ref="cpeDevice" @row-dblclick="rowdblclick" data-key="CPE" :rownumber="true" :url="cpeDeviceURL" :query-params="cpeQueryParams" :front-pagination="true">
								<template slot="toolbar">
									<div style="display: flex; flex-wrap: wrap; align-items: center;padding: 0 10px;">
										<el-query style="zoom: 0.8;" ref="cpelist" class="no-margin" type="normal" @query="queryCpeDevice" placeholder="<%=rb.getString("CPEXuLieHao")%>/<%=rb.getString("CPEName")%>"></el-query>
										
										<div style="padding: 5px 0 0 0;zoom: 0.9;">
											<el-checkbox v-model="cpeQueryParams.no_gps" true-label="" false-label="1"><%=rb.getString("WuJingWeiDuSheBei")%></el-checkbox>
											<span v-if="hasSAS" style="color: #a9a9a9;"><%=rb.getString("WuJingWeiDuSheBeiTiShi")%></span>
										</div>
									</div>
								</template>
								<el-table-column label="<%=rb.getString("CPEXuLieHao")%>" prop="serial_number" min-width="130"></el-table-column>
								<el-table-column label="<%=rb.getString("CPEName")%>" prop="host_name" min-width="100"></el-table-column>
								<el-table-column label="<%=rb.getString("WeiDu")%>" prop="latitude" min-width="90"></el-table-column>
								<el-table-column label="<%=rb.getString("JingDu")%>" prop="longitude" min-width="90"></el-table-column>
								<el-table-column width="45">
									<template slot-scope="scope">
										<div v-if="!sasEnable" class="el-icon el-icon-operation-settings" @click="optClick(scope.row,event)"></div>
										<div v-if="sasEnable" class="el-icon el-icon-operation-settings disabled"></div>
									</template>
								</el-table-column>
							</el-ctable>
						</el-tab-pane>
					</el-tabs>
				</div>
			</div>
		</div>
		<!-- topo map -->
		<div v-show="activeType=='group'" style="flex: auto; display: flex; flex-direction: column;position: relative;background: #fff;">
			<div :class="arrowClass" @click="groupShow = !groupShow"></div>
			<!-- 操作按钮 -->
			<div style="min-height: 30px;display: none;">
				<div style="padding: 8px;overflow: hidden;text-overflow:ellipsis;white-space:nowrap;width: 300px;color: #363B4E;font-size: 14px;">{{groupNames}}</div>
			</div>
			<div style="position: absolute; top: 20px; z-index: 1;right: 180px;display: flex;gap: 10px;">
				<span v-if="topoOverViewEnable" class="opt-group-wrapper">
					<div class="flex-bt-cls">
						<%=rb.getString("YinChanLieBiao")%>
						<el-checkbox v-model="overviewCheck" true-label="true" false-label="false" style="margin-left: 5px;"></el-checkbox>
					</div>
				</span>

				<span v-if="isFenceEnable" class="opt-group-wrapper">
					<div slot="reference" class="flex-bt-cls">
						<i :class="['el-icon', fenceEnable?'el-icon-operation-defaultBeta':'el-icon-operation-discard', 'margin-right-5']"></i> <%=rb.getString("DianZiWeiLan")%>
					</div>
					<el-popover :append-to-body="false" placement="bottom-end">
						<div slot="reference" class="flex-bt-cls">
							<i class="el-icon el-icon-down margin-left-5"></i>
						</div>
						<div style="width: 450px;">
							<div style="display: flex; justify-content: space-between; align-items: center; padding: 10px; border-bottom: 1px solid #ebeef5;">
								<span class="switch-cover" @click="changeFenceEnable">
									<el-switch 
										v-model="fenceEnable"
										active-color="#13ce66" 
										inactive-color="#ff4949" 
									></el-switch>
									<%=rb.getString("DianZiWeiLanKaiGuan")%>
								</span>
								<span>
									<i @click="drawFence" :class="['el-icon', 'el-icon-operation-add',fenceEnable?'':'disabled']" style="cursor: pointer;margin-right: 10px;"></i>
									<i onclick="document.body.click()" class="el-icon el-icon-close" style="cursor: pointer;"></i>
								</span>
							</div>
							<el-ctable height="350px"
								:data="fenceList"
								:filterstr="fenceQueryStr"
								:match-keys="['name']"
								:front-pagination="true"
								:pagination="true"
								stripe
								@row-dblclick="fenceRowDblClick"
							>
								<template slot="toolbar">
									<div style="display: flex; flex-wrap: wrap; align-items: center;padding: 0 10px;">
										<el-input style="width: 300px;"
											v-model="fenceQueryStr" 
											placeholder='<%=rb.getString("DianZiWeiLanMingCheng")%>'
											clearable
											suffix-icon="el-icon-search">
										</el-input>
									</div>
								</template>
								<el-table-column label="<%=rb.getString("QiYong")%>" prop="enable" width="70">
									<template slot-scope="scope">
										<el-switch 
											v-model="scope.row.enable"
											active-color="#13ce66" 
											inactive-color="#ff4949" 
											:disabled="!fenceEnable"
											class="switch-row"
											@change="fenceStatusChange(scope.row)"
										></el-switch>
									</template>
								</el-table-column>
								<el-table-column label="<%=rb.getString("DianZiWeiLanMingCheng")%>" prop="name"></el-table-column>
								<el-table-column label="" width="80">
									<template slot-scope="scope">
										<i @click="modifyFence(scope.row)" :class="['el-icon', 'el-icon-operation-edit', fenceEnable?'':'disabled']" style="cursor: pointer;margin-right: 10px;"></i>
										<i @click="removeFence(scope.row)" :class="['el-icon', 'el-icon-operation-delete', fenceEnable?'':'disabled']" style="cursor: pointer;"></i>
									</template>
								</el-table-column>
							</el-ctable>
						</div>
					</el-popover>
				</span>

				<span class="opt-group-wrapper">
					<el-popover :append-to-body="false">
						<div slot="reference" class="flex-bt-cls">
							<i class="el-icon el-icon-operation-settings margin-right-5"></i> <%=rb.getString("SheZhi")%>
						</div>
						<el-form style="width: 500px;padding: 15px;" class="form-item-bottom-15">
							<el-form-item>
								<el-checkbox-group v-model="statusForm.deviceType" style="margin-top: 10px;" @change="setStatus">
									<el-checkbox v-if="enbEnable" label="enb"><%=rb.getString("XiaoZhan")%></el-checkbox>
									<el-checkbox v-if="gnbEnable" label="gnb"><%=rb.getString("gNB")%></el-checkbox>
									<el-checkbox v-if="isGSMEnable" label="gsm">GSM</el-checkbox>
									<el-checkbox v-if="cpeEnable" label="cpe"><%=rb.getString("CPE")%></el-checkbox>
								</el-checkbox-group>
							</el-form-item>
							<el-form-item label="">
								<div>
									<%=rb.getString("SheBeiZaiXianZhuangTai")%>
									<el-radio v-show="false" v-model="statusType" label="deviceStatus" @change="statusTypeChange('deviceStatus')"><%=rb.getString("SheBeiZaiXianZhuangTai")%></el-radio>
								</div>
								<el-checkbox-group v-model="statusForm.deviceStatus" style="margin-left: 25px;margin-top: 10px;" @change="statusChange('deviceStatus')">
									<el-checkbox label="on"><%=rb.getString("ZaiXian")%></el-checkbox>
									<el-checkbox label="off"><%=rb.getString("LiXian")%></el-checkbox>
								</el-checkbox-group>
							</el-form-item>

							<el-form-item label="">
								<div>
									<%=rb.getString("ShiFouJiHuo")%>
									<el-radio v-show="false" v-model="statusType" label="activeStatus" @change="statusTypeChange('activeStatus')"><%=rb.getString("ShiFouJiHuo")%></el-radio>
								</div>
								<el-checkbox-group v-model="statusForm.activeStatus" style="margin-left: 25px;margin-top: 10px;" @change="statusChange('activeStatus')">
									<el-checkbox label="yes"><%=rb.getString("JiHuo")%></el-checkbox>
									<el-checkbox label="no"><%=rb.getString("QuJiHuo")%></el-checkbox>
								</el-checkbox-group>
							</el-form-item>

							<el-form-item label="" v-show="hasSAS">
								<div>
									<%=rb.getString("SASZhuangTaiQuanCheng")%>
									<el-radio v-show="false" v-model="statusType" label="sasStatus" @change="statusTypeChange('sasStatus')"><%=rb.getString("SASZhuangTaiQuanCheng")%></el-radio>
								</div>
								<el-checkbox-group v-model="statusForm.sasStatus" style="margin-left: 25px;margin-top: 10px;" @change="statusChange('sasStatus')">
									<el-checkbox label="Unregistered">Unregistered</el-checkbox>
									<el-checkbox label="Registered">Registered</el-checkbox>
									<el-checkbox label="Granted">Granted</el-checkbox>
									<el-checkbox label="Authorized">Authorized</el-checkbox>
								</el-checkbox-group>
							</el-form-item>
							<hr style="border: 1px solid #d8d8d8;border-bottom: none;"/>
							<el-form-item>
								<el-checkbox-group v-model="statusForm.ueStatus" style="margin-left: 25px;margin-top: 10px;" @change="ueStatusChange">
									<el-checkbox v-if="false" label="ue"><%=rb.getString("UEShu")%></el-checkbox>
									<el-checkbox label="mdt">MDT</el-checkbox>
								</el-checkbox-group>
							</el-form-item>
							<el-form-item>
								<el-checkbox-group v-model="statusForm.nameStatus" style="margin-left: 25px;margin-top: 10px;" @change="nameStatusChange">
									<el-checkbox label="snName">SN&Name</el-checkbox>
								</el-checkbox-group>
							</el-form-item>
						</el-form>
					</el-popover>
					<div class="flex-bt-cls" @click="measure">
						<i class="el-icon el-icon-ranging margin-right-5"></i> <%=rb.getString("CeJu")%>
					</div>
					<div class="flex-bt-cls" v-show="isLocal" @click="FullScreen">
						<i class="el-icon el-icon-fullscreen margin-right-5"></i> <%=rb.getString("QuanPin")%>
					</div>
					<div class="flex-bt-cls" @click="refreshTopoData">
						<i class="el-icon el-icon-common-refresh margin-right-5"></i><%=rb.getString("ShuaXin")%>
					</div>
				</span>
				<!-- KPI/UE -->
				<span class="opt-group-wrapper">
					<div class="flex-bt-cls" @click="kpiUeClick">
						<i class="el-icon el-icon-KPI-view margin-right-5"></i> KPI
					</div>
				</span>
			</div>
			<!-- 图例 -->
			<div class="topo-legend-cls" v-if="!topoOverViewEnable && !kpiPanelShow && statusType=='deviceStatus'">
				<div class="legend-title location-title"><%=rb.getString("WeiZhi")%></div>
				<div class="legend-title">
					<div style="font-weight: bold;text-align: left;">
						<%=rb.getString("XiaoZhan")%> <%=rb.getString("SheBeiZongShu")%>：{{deviceInfos.enbTotal}} <br/>
						<span v-if="gnbEnable"><%=rb.getString("gNB")%> <%=rb.getString("SheBeiZongShu")%>：{{deviceInfos.gnbTotal}} <br/></span>
						GSM <%=rb.getString("SheBeiZongShu")%>：{{deviceInfos.gsmTotal}} <br/>
						<span v-if="cpeEnable"><%=rb.getString("CPE")%> <%=rb.getString("SheBeiZongShu")%>：{{deviceInfos.cpeTotal}}</span>
					</div>
				</div>
				<div>
					<i class="el-icon el-icon-topo-enb online"></i> 
					<%=rb.getString("XiaoZhan")%> - <%=rb.getString("ZaiXian")%> 
					({{deviceInfos.enbOnlineCount}})
				</div>
				<div>
					<i class="el-icon el-icon-topo-enb offline"></i> 
					<%=rb.getString("XiaoZhan")%> - <%=rb.getString("LiXian")%> 
					({{deviceInfos.enbTotal - deviceInfos.enbOnlineCount}})
				</div>
				<div v-if="gnbEnable">
					<i class="el-icon el-icon_tpopo_5G online"></i> 
					<%=rb.getString("gNB")%> - <%=rb.getString("ZaiXian")%> 
					({{deviceInfos.gnbOnlineCount}})
				</div>
				<div v-if="gnbEnable">
					<i class="el-icon el-icon_tpopo_5G offline"></i> 
					<%=rb.getString("gNB")%> - <%=rb.getString("LiXian")%> 
					({{deviceInfos.gnbTotal - deviceInfos.gnbOnlineCount}})
				</div>
				<div v-if="isGSMEnable">
					<i class="el-icon el-icon_tpopo_2G online"></i> 
					GSM - <%=rb.getString("ZaiXian")%> 
					({{deviceInfos.gsmOnlineCount}})
				</div>
				<div v-if="isGSMEnable">
					<i class="el-icon el-icon_tpopo_2G offline"></i> 
					GSM - <%=rb.getString("LiXian")%> 
					({{deviceInfos.gsmTotal - deviceInfos.gsmOnlineCount}})
				</div>
				<div v-if="cpeEnable"><i class="el-icon el-icon-topo-cpe online"></i> <%=rb.getString("CPE")%> - <%=rb.getString("ZaiXian")%> ({{deviceInfos.cpeOnlineCount}})</div>
				<div v-if="cpeEnable"><i class="el-icon el-icon-topo-cpe offline"></i> <%=rb.getString("CPE")%> - <%=rb.getString("LiXian")%> ({{deviceInfos.cpeTotal - deviceInfos.cpeOnlineCount}})</div>
			</div>
			<!-- 图例 -->
			<div class="topo-legend-cls" v-if="!topoOverViewEnable && !kpiPanelShow && statusType=='activeStatus'">
				<div class="legend-title location-title"><%=rb.getString("WeiZhi")%></div>
				<div class="legend-title">
					<div style="font-weight: bold;text-align: left;">
						<%=rb.getString("XiaoZhan")%> <%=rb.getString("SheBeiZongShu")%>：{{deviceInfos.enbTotal}} <br/>
						<span v-if="gnbEnable"><%=rb.getString("gNB")%> <%=rb.getString("SheBeiZongShu")%>：{{deviceInfos.gnbTotal}} <br/></span>
						GSM <%=rb.getString("SheBeiZongShu")%>：{{deviceInfos.gsmTotal}} <br/>
						<span v-if="cpeEnable"><%=rb.getString("CPE")%> <%=rb.getString("SheBeiZongShu")%>：{{deviceInfos.cpeTotal}}</span>
					</div>
				</div>
				<div><i class="el-icon el-icon-topo-enb online"></i> <%=rb.getString("XiaoZhan")%> - <%=rb.getString("JiHuo")%> ({{deviceInfos.enbActiveCount}})</div>
				<div><i class="el-icon el-icon-topo-enb offline"></i> <%=rb.getString("XiaoZhan")%> - <%=rb.getString("QuJiHuo")%> ({{deviceInfos.enbTotal - deviceInfos.enbActiveCount}})</div>
				<div v-if="gnbEnable">
					<i class="el-icon el-icon_tpopo_5G online"></i> <%=rb.getString("gNB")%> - <%=rb.getString("JiHuo")%> ({{deviceInfos.gnbActiveCount}})
				</div>
				<div v-if="gnbEnable">
					<i class="el-icon el-icon_tpopo_5G offline"></i> <%=rb.getString("gNB")%> - <%=rb.getString("QuJiHuo")%> ({{deviceInfos.gnbTotal - deviceInfos.gnbActiveCount}})
				</div>
				<div v-if="isGSMEnable">
					<i class="el-icon el-icon_tpopo_2G online"></i> GSM - <%=rb.getString("JiHuo")%> ({{deviceInfos.gsmActiveCount}})
				</div>
				<div v-if="isGSMEnable">
					<i class="el-icon el-icon_tpopo_2G offline"></i> GSM - <%=rb.getString("QuJiHuo")%> ({{deviceInfos.gsmTotal - deviceInfos.gsmActiveCount}})
				</div>
			</div>
			<!-- 图例 -->
			<div class="topo-legend-cls" v-if="!topoOverViewEnable && !kpiPanelShow && statusType=='sasStatus'">
				<div class="legend-title location-title"><%=rb.getString("WeiZhi")%></div>
				<div class="legend-title">
					<div style="font-weight: bold;text-align: left;">
						<%=rb.getString("XiaoZhan")%> <%=rb.getString("SheBeiZongShu")%>：{{deviceInfos.enbTotal}} <br/>
						<span v-if="gnbEnable"><%=rb.getString("gNB")%> <%=rb.getString("SheBeiZongShu")%>：{{deviceInfos.gnbTotal}} <br/></span>
						<span v-if="isGSMEnable">GSM <%=rb.getString("SheBeiZongShu")%>：{{deviceInfos.gsmTotal}} <br/></span>
						<span v-if="cpeEnable"><%=rb.getString("CPE")%> <%=rb.getString("SheBeiZongShu")%>：{{deviceInfos.cpeTotal}}</span>
					</div>
				</div>
				<div>
					Unregistered ({{deviceInfos.unregistered}})
					<div style="padding: 8px 0;border-bottom: 1px dashed #e3e3e3;">
						<i class="el-icon el-icon-topo-enb offline"></i> <%=rb.getString("XiaoZhan")%>({{deviceInfos.enbunregistered}})
						<span v-if="gnbEnable"><i class="el-icon el-icon_tpopo_5G offline"></i> <%=rb.getString("gNB")%>({{deviceInfos.gnbunregistered}})</span>
						<span v-if="isGSMEnable"><i class="el-icon el-icon_tpopo_2G offline"></i> GSM({{deviceInfos.gsmunregistered}})</span>
						<span v-if="cpeEnable"><i class="el-icon el-icon-topo-cpe offline"></i> <%=rb.getString("CPE")%>({{deviceInfos.cpeunregistered}})</span>
					</div>
				</div>
				<div>
					Registered ({{deviceInfos.registered}})
					<div style="padding: 8px 0;border-bottom: 1px dashed #e3e3e3;">
						<i class="el-icon el-icon-topo-enb registed"></i> <%=rb.getString("XiaoZhan")%>({{deviceInfos.enbregistered}})
						<span v-if="gnbEnable"><i class="el-icon el-icon_tpopo_5G registed"></i> <%=rb.getString("gNB")%>({{deviceInfos.gnbregistered}})</span>
						<span v-if="isGSMEnable"><i class="el-icon el-icon_tpopo_2G registed"></i> GSM({{deviceInfos.gsmregistered}})</span>
						<span v-if="cpeEnable"><i class="el-icon el-icon-topo-cpe registed"></i> <%=rb.getString("CPE")%>({{deviceInfos.cperegistered}})</span>
					</div>
				</div>
				<div>
					Granted ({{deviceInfos.granted}})
					<div style="padding: 8px 0;border-bottom: 1px dashed #e3e3e3;">
						<i class="el-icon el-icon-topo-enb granted"></i> <%=rb.getString("XiaoZhan")%>({{deviceInfos.enbgranted}})
						<span v-if="gnbEnable"><i class="el-icon el-icon_tpopo_5G granted"></i> <%=rb.getString("gNB")%>({{deviceInfos.gnbgranted}})</span>
						<span v-if="isGSMEnable"><i class="el-icon el-icon_tpopo_2G granted"></i> GSM({{deviceInfos.gsmgranted}})</span>
						<span v-if="cpeEnable"><i class="el-icon el-icon-topo-cpe granted"></i> <%=rb.getString("CPE")%>({{deviceInfos.cpegranted}})</span>
					</div>
				</div>
				<div>
					Authorized ({{deviceInfos.authorized}})
					<div style="padding: 8px 0;">
						<i class="el-icon el-icon-topo-enb authed"></i> <%=rb.getString("XiaoZhan")%>({{deviceInfos.enbauthorized}})
						<span v-if="gnbEnable"><i class="el-icon el-icon_tpopo_5G authed"></i> <%=rb.getString("gNB")%>({{deviceInfos.gnbauthorized}})</span>
						<span v-if="isGSMEnable"><i class="el-icon el-icon_tpopo_2G authed"></i> GSM({{deviceInfos.gsmauthorized}})</span>
						<span v-if="cpeEnable"><i class="el-icon el-icon-topo-cpe authed"></i> <%=rb.getString("CPE")%>({{deviceInfos.cpeauthorized}})</span>
					</div>
				</div>
			</div>
			<!-- topo 地图 -->
			<div style="flex: auto;overflow: auto;position: relative;" :class="sasSwitchCls">
				<div id="map" style="width: 100%;min-height: 100%;background: #A1D4E0;"></div>
				
				<!-- 搜索切换按钮 -->
				<div class="toggle-btn search-toggle-btn" 
					 :class="{active: searchPanelShow}"
					 @click="toggleSearchPanel" 
					 :title="searchPanelShow ? '<%=rb.getString("GuanBi")%>' : '<%=rb.getString("SouSuo")%>'">
					<i class="el-icon el-icon-common-search"></i>
				</div>
				
				<!-- 左侧搜索面板 -->
				<div class="search-panel" :class="{show: searchPanelShow}">
					<div class="search-panel-header">
						<h3><%=rb.getString("SouSuo")%></h3>
						<span class="close-btn" @click="searchPanelShow = false">×</span>
					</div>
					<div class="search-input">
						<el-input 
							style="width: 90%;"
							v-model="searchKeyword" 
							placeholder="<%=rb.getString("HostName")%>/<%=rb.getString("SheBeiMingCheng")%>/<%=rb.getString("IPDiZhi")%>" 
							@input="handleSearch"
							clearable>
							<i slot="prefix" class="el-input__icon el-icon-search"></i>
						</el-input>
					</div>
					<div class="search-results">
						<div v-if="searchResults.length === 0 && searchKeyword" style="padding: 20px; text-align: center; color: #909399;">
							<%=rb.getString("MeiShuJu")%>
						</div>
						<div 
							v-for="node in searchResults" 
							:key="node.cellCode || node.code"
							class="search-result-item"
							@click="locateNodeSmart(node)">
							<div class="node-code"><%=rb.getString("SheBeiMingCheng")%>: {{node.code}}</div>
							<div class="node-info">
								<div v-if="node.name"><%=rb.getString("HostName")%>: {{node.name}}</div>
								<div v-if="node.ip"><%=rb.getString("IPDiZhi")%>: {{node.ip}}</div>
								<div><%=rb.getString("ZuMing")%>: {{node.groupName}}</div>
								<div><%=rb.getString("Type")%>: {{getNodeTypeText(node.type)}}</div>
							</div>
						</div>
					</div>
				</div>
				
				<!-- 右侧重叠节点列表面板 -->
				<div class="cluster-panel" :class="{show: clusterPanelShow}">
					<div class="cluster-panel-header">
						<h3><%=rb.getString("ChongDieJieDian")%> ({{clusterNodes.length}})</h3>
						<span class="close-btn" @click="clusterPanelShow = false">×</span>
					</div>
					<div class="cluster-panel-body">
						<div 
							v-for="(node, index) in clusterNodes" 
							:key="node.cellCode || index"
							class="cluster-node-item"
							@click="locateNode(node, true)">
							<div class="node-header">
								<span class="node-code"><%=rb.getString("SheBeiMingCheng")%>: {{node.code}}</span>
								<span class="node-status" :class="node.online === 'on' ? 'online' : 'offline'">
									{{node.online === 'on' ? '<%=rb.getString("ZaiXian")%>' : '<%=rb.getString("LiXian")%>'}}
								</span>
							</div>
							<div class="node-details">
								<div v-if="node.name"><%=rb.getString("HostName")%>: {{node.name}}</div>
								<div><%=rb.getString("Type")%>: {{getNodeTypeText(node.type)}}</div>
								<div v-if="node.groupName"><%=rb.getString("ZuMing")%>: {{node.groupName}}</div>
							</div>
						</div>
					</div>
				</div>
			</div>

			<!-- Zed 大屏 -->
			<div v-if="topoOverViewEnable && overviewCheck == 'false'" class="all-device-wrap">
				<el-table :data="overViewList" border stripe class="border-overview">
					<el-table-column type="index" width="30"></el-table-column>
					<el-table-column label="Area" prop="area"></el-table-column>
					<el-table-column label="GSM Total" prop="btsTotal" align="center"></el-table-column>
					<el-table-column label="GSM Online" prop="btsOnline" align="center"></el-table-column>
					<el-table-column label="eNB Total" prop="enbTotal" align="center"></el-table-column>
					<el-table-column label="eNB Online" prop="enbOnline" align="center"></el-table-column>
					<el-table-column label="gNB Total" prop="gnbTotal" align="center"></el-table-column>
					<el-table-column label="gNB Online" prop="gnbOnline" align="center"></el-table-column>
					<el-table-column label="Online Total" prop="onlineTotal" align="center"></el-table-column>
					<el-table-column label="Total" prop="total" align="center"></el-table-column>
				</el-table>
			</div>

			<!-- 信息区域 -->
			<div :class="infoPanelCls">
				<div class="panel-arrow" @click="hidePanel">
					<i class="el-icon el-icon-down"></i>
				</div>
				<div v-show="infoShow && infoForm.type=='enb'">
					<div class="info-panel-title">
                        <%=rb.getString("XinXi")%>
                        <span class="el-icon el-icon-operation-settings" @click="showTopoSettingPanel('enb')"></span>
                    </div>
					<div class="info-item-cls" label="<%=rb.getString("XiaoZhanBianMa")%>" style="margin-top: 10px;">{{infoForm.sn}}</div>
					<div class="info-item-cls" label="<%=rb.getString("HostName")%>">{{infoForm.name}}</div>
					<div class="info-item-cls" label="<%=rb.getString("IPDiZhi")%>">{{infoForm.ip}}</div>
					<div class="info-item-cls" label="<%=rb.getString("SheBeiZu")%>">{{infoForm.groupName}}</div>
					<div class="info-item-cls" label="<%=rb.getString("ShiFouJiHuo")%>" v-html="activeFmt(infoForm.active, infoForm)"></div>
					<div class="info-item-cls" label="<%=rb.getString("GPSWeiZhi")%>">
						{{infoForm.lat}}, {{infoForm.lon}} 
						<span class="el-icon el-icon-operation-edit" v-show="!sasEnable" style="margin-left: 5px;" @click="modifyInfoGPS"></span>
					</div>
					<div class="info-item-cls" label="<%=rb.getString("GaoJingJiBie")%>" v-html="alarmFmt(infoForm.alarm,infoForm)"></div>
					<div class="info-item-cls" label="<%=rb.getString("MMEZhuangTai")%>" style="position: relative;">
						<div v-if="['','NULL','null',null,undefined].includes(infoForm.NEW_MME_STATUS)">
							<span v-html="oldMmeStatusFmt(infoForm, infoForm.mmeStatus, 'left')" ></span>
                            <el-popover title="All MME" popper-class="mmePopoverClass" v-if="oldMmeStatusFmt(infoForm, infoForm.mmeStatus, 'list').length > 0" :append-to-body="false">
                                <span style="color:#4d84ff;cursor:pointer;" slot="reference" v-if="oldMmeStatusFmt(infoForm, infoForm.mmeStatus, 'list').length > 0">
                                    [ <span v-html="oldMmeStatusFmt(infoForm, infoForm.mmeStatus, 'right')"></span> ]
                                </span>
                                <div class="mme-list">
                                    <i class="el-icon el-icon-close" style="position: absolute; top: -30px; right: 0px;" onclick="document.body.click()"></i>
                                    <div class="mme-list-item" v-for="item in oldMmeStatusFmt(infoForm, infoForm.mmeStatus, 'list')">
                                        <span v-if="item.status == '1'" style="margin-top:10px;" class="el-icon el-icon-status-MME greenIcon"></span>
                                        <span v-if="item.status == '0'" style="margin-top:10px;" class="el-icon el-icon-status-MME redIcon"></span> 
                                        <div class="mme-info">
                                            <span>MME IP : {{item.mmeIp}}</span>
                                            <span v-if="item.status==='0'"><%=rb.getString("MMEZhuangTai")%> : <%= rb.getString("MMEWeiLianJie")%></span>
                                            <span v-if="item.status=='1'"><%=rb.getString("MMEZhuangTai")%> : <%= rb.getString("MMEYiLianJie")%></span>
                                            <span v-if="item.status=='2'"><%=rb.getString("MMEZhuangTai")%> : --</span>
                                            <span v-if="item.status==='' || item.status===null"><%=rb.getString("MMEZhuangTai")%> : </span>
                                        </div>
                                    </div>
                                </div>
                            </el-popover>
						</div>
						<div v-if="!['','NULL','null',null,undefined].includes(infoForm.NEW_MME_STATUS)" style="display: flex;align-items: center;">
							<span v-html="mmesStatusFmt(infoForm.NEW_MME_STATUS)" ></span>
							<el-popover title="All MME" popper-class="mmePopoverClass" :append-to-body="false">
                                <span style="color:#4d84ff;cursor:pointer;" slot="reference">
                                    [ {{parseMMEOnNum(infoForm.NEW_MME_STATUS)}}/{{parseMME(infoForm.NEW_MME_STATUS).length}} ]
                                </span>
                                <div class="mme-list">
                                    <i class="el-icon el-icon-close" style="position: absolute; top: -30px; right: 0px;" onclick="document.body.click()"></i>
                                    <div class="mme-list-item" v-for="item in parseMME(infoForm.NEW_MME_STATUS)">
                                        <span v-if="item.status == '1'" class="el-icon el-icon-status-MME greenIcon"></span>
                                        <span v-if="item.status == '0'" class="el-icon el-icon-status-MME redIcon"></span> 
                                        <div class="mme-info">
                                            <span>MME IP : {{item.mmeIp}}</span>
                                            <span v-if="item.status==='0'"><%=rb.getString("MMEZhuangTai")%> : Disconnected</span>
                                            <span v-if="item.status=='1'"><%=rb.getString("MMEZhuangTai")%> : Connected</span>
                                            <span v-if="item.status=='2'"><%=rb.getString("MMEZhuangTai")%> : --</span>
                                            <span v-if="item.status==='' || item.status===null"><%=rb.getString("MMEZhuangTai")%> : </span>
                                            <span><%=rb.getString("PLMN")%> : {{item.plmnId}}</span>
                                        </div>
                                    </div>
                                </div>
                            </el-popover>
						</div>
					</div>
					<div v-if="false" class="info-item-cls" label="<%=rb.getString("RSMME")%>" v-html="mmeIPfmt(infoForm.mmeStatus,infoForm)"></div>
					<div class="info-item-cls" label="<%=rb.getString("SASKaiGuan")%>" v-show="hasSAS" v-html="sasEnableFmt(infoForm.mmeStatus,infoForm)"></div>
					<div class="info-item-cls" label="<%=rb.getString("SASZhuangTai")%>" v-show="hasSAS" v-html="sasStateFmt(infoForm.sasState,infoForm)"></div>
					<div class="info-item-cls" label="<%=rb.getString("UEShu")%>">{{infoForm.ueCount}}</div>
					<div class="info-item-cls" label="<%=rb.getString("PCI2")%>">{{infoForm.pci}}</div>
					<div class="info-item-cls" label="<%=rb.getString("ZuiDaBanJin")%>">{{infoForm.radius}}</div>
					<div class="info-item-cls" label="<%=rb.getString("ZuiXiaoBanJin")%>">{{infoForm.minRadius}}</div>
				</div>
				<div v-show="infoShow && infoForm.type=='gnb'">
					<div class="info-panel-title">
                        <%=rb.getString("XinXi")%>
                        <span class="el-icon el-icon-operation-settings" @click="showTopoSettingPanel('gnb')"></span>
                    </div>
					<div class="info-item-cls" label="<%=rb.getString("XiaoZhanBianMa")%>" style="margin-top: 10px;">{{infoForm.sn}}</div>
					<div class="info-item-cls" label="<%=rb.getString("HostName")%>">{{infoForm.name}}</div>
					<div class="info-item-cls" label="<%=rb.getString("IPDiZhi")%>">{{infoForm.ip}}</div>
					<div class="info-item-cls" label="<%=rb.getString("SheBeiZu")%>">{{infoForm.groupName}}</div>
					<div class="info-item-cls" label="<%=rb.getString("ShiFouJiHuo")%>" v-html="activeFmt(infoForm.active, infoForm)"></div>
					<div class="info-item-cls" label="<%=rb.getString("GPSWeiZhi")%>">
						{{infoForm.lat}}, {{infoForm.lon}} 
						<span class="el-icon el-icon-operation-edit" v-show="!sasEnable" style="margin-left: 5px;" @click="modifyInfoGPS"></span>
					</div>
					<div class="info-item-cls" label="<%=rb.getString("GaoJingJiBie")%>" v-html="alarmFmt(infoForm.alarm,infoForm)"></div>
					<div class="info-item-cls" label="<%=rb.getString("SASKaiGuan")%>" v-show="hasSAS" v-html="sasEnableFmt(infoForm.mmeStatus,infoForm)"></div>
					<div class="info-item-cls" label="<%=rb.getString("SASZhuangTai")%>" v-show="hasSAS" v-html="sasStateFmt(infoForm.sasState,infoForm)"></div>
					<div class="info-item-cls" label="<%=rb.getString("UEShu")%>">{{infoForm.ueCount}}</div>
				</div>
				<div v-show="infoShow && infoForm.type=='gsm'">
					<div class="info-panel-title">
                        <%=rb.getString("XinXi")%>
                        <span class="el-icon el-icon-operation-settings" @click="showTopoSettingPanel('gsm')"></span>
                    </div>
					<div class="info-item-cls" label="<%=rb.getString("XiaoZhanBianMa")%>" style="margin-top: 10px;">{{infoForm.sn}}</div>
					<div class="info-item-cls" label="<%=rb.getString("HostName")%>">{{infoForm.name}}</div>
					<div class="info-item-cls" label="<%=rb.getString("IPDiZhi")%>">{{infoForm.ip}}</div>
					<div class="info-item-cls" label="<%=rb.getString("SheBeiZu")%>">{{infoForm.groupName}}</div>
					<div class="info-item-cls" label="<%=rb.getString("ShiFouJiHuo")%>" v-html="activeFmt(infoForm.active, infoForm)"></div>
					<div class="info-item-cls" label="<%=rb.getString("GPSWeiZhi")%>">
						{{infoForm.lat}}, {{infoForm.lon}} 
						<span class="el-icon el-icon-operation-edit" v-show="!sasEnable" style="margin-left: 5px;" @click="modifyInfoGPS"></span>
					</div>
					<div class="info-item-cls" label="<%=rb.getString("GaoJingJiBie")%>" v-html="alarmFmt(infoForm.alarm,infoForm)"></div>
					<div class="info-item-cls" label="<%=rb.getString("SASKaiGuan")%>" v-show="hasSAS" v-html="sasEnableFmt(infoForm.mmeStatus,infoForm)"></div>
					<div class="info-item-cls" label="<%=rb.getString("SASZhuangTai")%>" v-show="hasSAS" v-html="sasStateFmt(infoForm.sasState,infoForm)"></div>
					<div class="info-item-cls" label="<%=rb.getString("UEShu")%>">{{infoForm.ueCount}}</div>
				</div>
				<div v-show="infoShow && infoForm.type=='cpe'">
					<div class="info-panel-title">
                        <%=rb.getString("XinXi")%>
                        <span class="el-icon el-icon-operation-settings" @click="showTopoSettingPanel('cpe')"></span>
                    </div>
					<div class="info-item-cls" label="CPE SN" style="margin-top: 10px;">{{infoForm.sn}}</div>
					<div class="info-item-cls" label="eNB SN">{{infoForm.relaEnb}}</div>
					<div class="info-item-cls" label="<%=rb.getString("CPEName")%>" style="word-break: break-all;align-items: baseline;">{{infoForm.name}}</div>
					<div class="info-item-cls" label="<%=rb.getString("IMSI")%>">{{infoForm.imsi}}</div>
					<div class="info-item-cls" label="<%=rb.getString("IPDiZhi")%>">{{infoForm.ip}}</div>
					<div class="info-item-cls" label="<%=rb.getString("MACDiZhi")%>">{{infoForm.mac}}</div>
					<div class="info-item-cls" label="<%=rb.getString("GPSWeiZhi")%>">
						{{infoForm.lat}}, {{infoForm.lon}}
						<span class="el-icon el-icon-operation-edit" v-show="!sasEnable" style="margin-left: 5px;" @click="modifyInfoGPS"></span>
					</div>
					<div class="info-item-cls" label="<%=rb.getString("GaoJingJiBie")%>" v-html="alarmFmt(infoForm.alarm,infoForm)"></div>
					<div class="info-item-cls" label="<%=rb.getString("WeiXingXinHao")%>" style="position: relative;" v-html="rsrpFmt(infoForm)"></div>
					<div class="info-item-cls" label="<%=rb.getString("SheBeiZu")%>">{{infoForm.groupName}}</div>
					<div class="info-item-cls" label="<%=rb.getString("SASKaiGuan")%>" v-show="hasSAS" v-html="sasEnableFmt(infoForm.sasEnable, infoForm)"></div>
					<div class="info-item-cls" label="<%=rb.getString("SASZhuangTai")%>" v-show="hasSAS" v-html="sasStateFmt(infoForm.sasState, infoForm)"></div>
				</div>
			</div>
		</div>

		<!-- Site Map -->
		<div v-show="activeType=='site'" style="flex: auto; display: flex; flex-direction: column;position: relative;background: #fff;" :class="{'hidden-site-name': !siteNameShow}">
			<!-- Site 地图 -->
			<div style="flex: auto;overflow: auto;">
				<div id="site_map" style="width: 100%;min-height: 100%;background: #A1D4E0;"></div>
			</div>

			<div style="position: absolute; top: 20px; right: 180px;padding: 5px;background: #fff;z-index: 1;border-radius: 5px;display:flex;">
				<el-popover :append-to-body="false">
					<div slot="reference" class="flex-bt-cls">
						<i class="el-icon el-icon-operation-settings margin-right-5"></i> <%=rb.getString("SheZhi")%>
					</div>
					<div style="padding: 15px; border-bottom: 1px solid #E9EdF9;">
						<el-checkbox :indeterminate="isIndeterminate" v-model="siteCheckAll" @change="siteCheckAllChange">All</el-checkbox>
					</div>
					<el-form style="padding: 0 15px;width: 100px;">
						<el-form-item style="margin-bottom: 0px;">
							<el-checkbox-group v-model="siteDeviceType" @change="siteCheckChange" class="no-margin" style="margin-top: 10px;">
								<el-checkbox v-if="isGSMEnable" label="gsm">GSM</el-checkbox>
								<el-checkbox v-if="enbEnable" label="enb"><%=rb.getString("XiaoZhan")%></el-checkbox>
								<el-checkbox v-if="gnbEnable" label="gnb"><%=rb.getString("gNB")%></el-checkbox>
							</el-checkbox-group>
						</el-form-item>
					</el-form>
					<div style="padding: 10px 15px 15px; border-top: 1px solid #E9EdF9;">
						<el-checkbox v-model="siteNameShow">Site Name</el-checkbox>
					</div>

					<div v-if="false" style="padding: 10px 15px;">
						<el-button type="primary" size="mini" style="min-width:60px;"><%=rb.getString("QueDing")%></el-button>
						<el-button size="mini" style="min-width:60px;padding: 0 10px;"><%=rb.getString("QuXiao")%></el-button>
					</div>
				</el-popover>
				<div class="flex-bt-cls" @click="measureSite">
					<i class="el-icon el-icon-ranging margin-right-5"></i> <%=rb.getString("CeJu")%>
				</div>
				<div class="flex-bt-cls" v-show="isLocal" @click="FullScreen">
					<i class="el-icon el-icon-fullscreen margin-right-5"></i> <%=rb.getString("QuanPin")%>
				</div>
				<div class="flex-bt-cls" @click="refreshSiteMap">
					<i class="el-icon el-icon-common-refresh margin-right-5"></i><%=rb.getString("ShuaXin")%>
				</div>
			</div>

			<!-- 图例 -->
			<div class="topo-legend-cls" v-if="activeType=='site'" style="top: 20px;">
				<div class="legend-title location-title">
					<%=rb.getString("WeiZhi")%>
				</div>
				<div class="legend-title">
					<div style="font-weight: bold;text-align: left;">
						<%=rb.getString("ZhanDianMingCheng")%> <%=rb.getString("SheBeiZongShu")%>：{{allSiteLatlonNodes.length}} <br/>
					</div>
				</div>
				<!-- 展示站点状态（Normal\Abnormal\Offline）类型及统计数 -->
				<div style="margin-top: 15px;flex-direction: row;" class="node-item-leaf normal">
					<i class="el-icon el-icon_tpopo_site" style="font-size: 14px;margin-right: 8px;"></i>
					Normal ({{siteStatusCount.normalCount}})
				</div>
				<div class="node-item-leaf abnormal" style="flex-direction: row;">
					<i class="el-icon el-icon_tpopo_site" style="font-size: 14px;margin-right: 8px;"></i>
					Abnormal ({{siteStatusCount.abnormalCount}})
				</div>
				<div class="node-item-leaf offline" style="flex-direction: row;">
					<i class="el-icon el-icon_tpopo_site" style="font-size: 14px;margin-right: 8px;"></i>
					Offline ({{siteStatusCount.offlineCount}})
				</div>
			</div>

			<!-- 信息区域 -->
			<div :class="sitePanelCls" style="height: 100%;top: 0px;display: flex;flex-direction: column;">
				<div class="site-pane-cls">
					<el-tabs v-if="siteType == 'view'" size="mini" v-model="siteTab">
						<el-tab-pane label='<%=rb.getString("XinXi")%>' name="siteName"> 
						</el-tab-pane>
						<el-tab-pane label='<%=rb.getString("SheBeiLieBiao")%>' name="deviceList"> 
						</el-tab-pane>
					</el-tabs>

					<span v-if="siteType == 'view' && false" style="font-weight: bold;color: #7A7992;">{{currentSiteRow.siteName}}</span>
					<span v-if="siteType == 'add'" style="font-weight: bold;color: #7A7992;"><%=rb.getString("TianJia")%></span>
					<span style="display: flex;align-items: center;">
						<i v-if="siteType == 'view' && siteTab == 'deviceList'" @click="toAddSiteDevice" class="el-icon el-icon-operation-add" style="margin-right: 10px;"></i>
						<i @click="siteInfoShow = false" class="el-icon el-icon-close"></i>
					</span>
				</div>

				<!-- add Site -->
				<div v-show="siteType == 'add'" style="padding: 15px;flex: auto;overflow: auto;">
					<el-form ref="addForm" :model="siteAddForm" :rules="addRule" label-position="top">
						<el-form-item label='<%=rb.getString("ZhanZhiMingCheng")%>' prop="siteName" required>
							<el-input v-model="siteAddForm.siteName" style="width: 100%;" maxlength="50"></el-input>
						</el-form-item>
						<el-form-item label='<%=rb.getString("WeiZhi")%>' prop="latitude" required>
							<div style="position: absolute;right: 1px;top: -25px;">
								<span @click="addPickOnMap" style="color: #689CF3;cursor: pointer;display: flex;align-items: center;">
									<i class="el-icon el-icon-draw"></i> 
									<span style="text-decoration: underline;margin-left: 5px;">Pick on MAP</span>
								</span>
							</div>
							<el-input v-model="siteAddForm.latitude" style="width: 100%;">
								<template slot="prepend">
									<div style="width: 60px;"><%=rb.getString("WeiDu")%></div>
								</template>
							</el-input>
						</el-form-item>
						<el-form-item prop="longitude" class="no-padding-top">
							<el-input v-model="siteAddForm.longitude" style="width: 100%;margin-top: 20px;">
								<template slot="prepend">
									<div style="width: 60px;"><%=rb.getString("JingDu")%></div>
								</template>
							</el-input>
						</el-form-item>
						<el-form-item label='<%=rb.getString("TianJiaLeiXing")%>'>
							<el-checkbox-group v-model="siteAddDeviceType" @change="siteAddDeviceTypeChange" size="small">
								<el-checkbox v-if="isGSMEnable" label="GSM" border></el-checkbox>
								<el-checkbox v-if="enbEnable" label="eNB" border></el-checkbox>
								<el-checkbox v-if="gnbEnable" label="gNB" border></el-checkbox>
							</el-checkbox-group>
						</el-form-item>
						<el-form-item label='<%=rb.getString("SheBeiLieBiaoBiaoTi")%>'>
							<el-ctable ref="addTb" style="height: 260px;"
								row-key="small_cell_code"
								:url="addDeviceURL"
								:query-params="addQuery"
								:show-pager="false"
								@selection-change="addSelectionChange"
							>
								<template slot="toolbar">
									<el-query @query="addQueryClick" style="zoom: 0.8;" class="width-fitcontent" type="normal" placeholder='<%=rb.getString("XiaoZhanBianMa")%>/<%=rb.getString("HostName")%>'></el-query>
								</template>
								<el-table-column type="selection" :reserve-selection="true"></el-table-column>
								<el-table-column width="35">
									<template slot-scope="scope">
										<div style="zoom: 0.98;" :class="{
											'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
											'':scope.row.have_connected==2,
											'conn_exc':scope.row.connection_status=='Exception',
											'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
									</template>
								</el-table-column>
								<el-table-column label='<%=rb.getString("XiaoZhanBianMa") %>' prop="serial_number"></el-table-column>
								<el-table-column label='<%=rb.getString("HostName")%>' prop="host_name"></el-table-column>
								<el-table-column label='<%=rb.getString("Type")%>' prop="stationType" width="50"></el-table-column>
							</el-ctable>
							<el-checkbox v-model="siteAddForm.locationConsistent"
								:true-label="true"
								:false-label="false"
							>
								Device and site locations are synchronized
							</el-checkbox>
						</el-form-item>
					</el-form>
				</div>

				<!-- Info Site -->
				<!-- Information View -->
				<div v-show="siteType == 'view' && siteTab=='siteName'" style="padding: 15px;flex: auto;overflow: auto;">
					<ul class="node-ul information-ul">
						<li class="space-between"><span class="node-label">Site Name</span> 
							<span style="word-break: break-all;">{{currentSiteRow.siteName}}</span>
						</li> 
						<li class="space-between"><span class="node-label">Status</span>
							<span v-if="currentSiteRow.status == '1'" class="node-item-leaf normal" style="display: flex;align-items: center;flex-direction: row;">
								<i class="el-icon el-icon_tpopo_site" style="font-size: 12px;margin-right: 8px;"></i> 
								Normal
							</span>
							<span v-if="currentSiteRow.status == '2'" class="node-item-leaf abnormal" style="display: flex;align-items: center;flex-direction: row;">
								<i class="el-icon el-icon_tpopo_site" style="font-size: 12px;margin-right: 8px;"></i>
								Abnormal
							</span>
							<span v-if="currentSiteRow.status == '3'" class="node-item-leaf offline" style="display: flex;align-items: center;flex-direction: row;">
								<i class="el-icon el-icon_tpopo_site" style="font-size: 12px;margin-right: 8px;"></i>
								Offline
							</span>
						</li> 
						<li class="space-between"><span class="node-label">GPS Position</span>
							<span>
								{{currentSiteRow.latitude}}, {{currentSiteRow.longitude}} 
								<span class="el-icon el-icon-operation-edit" v-show="!sasEnable" style="margin-left: 5px;color:#4d84ff;" @click="modifySiteInfoGPS"></span>
							</span>
						</li> 
						<li class="space-between"><span class="node-label">eNB Count</span> {{currentSiteRow.enbCount || 0}} </li>
						<li class="space-between"><span class="node-label">gNB Count</span> {{currentSiteRow.gnbCount || 0}} </li>
						<li v-if="isGSMEnable" class="space-between"><span class="node-label">GSM Count</span> {{currentSiteRow.gsmCount || 0}} </li>
						<li v-if="enbEnable" class="space-between"><span class="node-label">eNB Throughput(Mbps)</span> {{currentSiteRow.enbThroughput || 0}} </li>
						<li v-if="gnbEnable" class="space-between"><span class="node-label">gNB Throughput(Mbps)</span> {{currentSiteRow.gnbThroughput || 0}} </li>
					<ul>
				</div>

				<div v-show="siteType == 'view' && siteTab == 'deviceList'" style="padding: 15px;flex: auto;overflow: auto;">
					<el-ctable ref="viewTb" size="small"
						row-key="serialNumber"
						:url="viewDeviceURL"
						:query-params="viewQuery"
						:show-pager="false"
					>
						<template slot="toolbar">
							<el-query @query="viewQueryClick" style="zoom: 0.8;" class="width-fitcontent" type="normal" placeholder='<%=rb.getString("XiaoZhanBianMa")%>/<%=rb.getString("HostName")%>'></el-query>
						</template>
						<el-table-column width="40">
							<template slot-scope="scope">
								<span @click="siteOptClick(scope.row, event)" v-clickoutside="handerClose" class="el-icon el-icon-operation-more"></span>
							</template>
						</el-table-column>
						<el-table-column prop="connection_status" width="50">
							<template slot-scope="scope">
								<div :class="{
									'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
									'':scope.row.have_connected==2,
									'conn_exc':scope.row.connection_status=='Exception',
									'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
							</template>
						</el-table-column>
						<el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' prop="serial_number"></el-table-column>
						<el-table-column label='<%=rb.getString("HostName")%>' prop="host_name"></el-table-column>
						<el-table-column label='<%=rb.getString("Type")%>' prop="stationType" width="50"></el-table-column>
					</el-ctable>

					<el-cmenu ref="siteMenu" :data="siteMenus" @click="menuClick"></el-cmenu>
				</div>

				<div v-if="siteType == 'add'" style="padding: 15px;">
					<el-button type="primary" @click="addSiteSubmit"><%=rb.getString("QueDing")%></el-button>
					<el-button @click="siteInfoShow = false"><%=rb.getString("QuXiao")%></el-button>
				</div>
			</div>

			<!-- Details -->
			<div v-show="deviceDetailShow" class="detail-panel">
				<div class="detail-header">
					<span style="font-weight: bold;color: #7A7992;">
						<%=rb.getString("XinXi")%>
						<span style="color: rgba(0,0,0,.16)"> ({{detailForm.code}}) </span>
					</span>
					<i @click="deviceDetailShow = false" class="el-icon el-icon-close"></i>
				</div>
				<div class="detail-body">
					<div class="info-item-cls" label="<%=rb.getString("HostName")%>">{{detailForm.name}}</div>
					<div class="info-item-cls" label="<%=rb.getString("IPDiZhi")%>">{{detailForm.ip}}</div>
					<div class="info-item-cls" label="<%=rb.getString("SheBeiZu")%>">{{detailForm.groupName}}</div>
					<div class="info-item-cls" label="<%=rb.getString("ShiFouJiHuo")%>" v-html="activeFmt(detailForm.active, detailForm)"></div>
					<div class="info-item-cls" label="<%=rb.getString("GPSWeiZhi")%>">
						{{detailForm.lat}}, {{detailForm.lon}} 
						<span class="el-icon el-icon-operation-edit" v-show="!sasEnable" style="margin-left: 5px;" @click="toInfoModifyGPS"></span>
					</div>
					<div class="info-item-cls" label="<%=rb.getString("GaoJingJiBie")%>" v-html="alarmFmt(detailForm.alarm, detailForm)"></div>
					<div class="info-item-cls" label="<%=rb.getString("MMEZhuangTai")%>" style="position: relative;">
						<div v-if="['','NULL','null',null,undefined].includes(infoForm.NEW_MME_STATUS)">
							<span v-html="oldMmeStatusFmt(infoForm, infoForm.mmeStatus, 'left')" ></span>
                            <el-popover title="All MME" popper-class="mmePopoverClass" v-if="oldMmeStatusFmt(infoForm, infoForm.mmeStatus, 'list').length > 0" :append-to-body="false">
                                <span style="color:#4d84ff;cursor:pointer;" slot="reference" v-if="oldMmeStatusFmt(infoForm, infoForm.mmeStatus, 'list').length > 0">
                                    [ <span v-html="oldMmeStatusFmt(infoForm, infoForm.mmeStatus, 'right')"></span> ]
                                </span>
                                <div class="mme-list">
                                    <i class="el-icon el-icon-close" style="position: absolute; top: -30px; right: 0px;" onclick="document.body.click()"></i>
                                    <div class="mme-list-item" v-for="item in oldMmeStatusFmt(infoForm, infoForm.mmeStatus, 'list')">
                                        <span v-if="item.status == '1'" style="margin-top:10px;" class="el-icon el-icon-status-MME greenIcon"></span>
                                        <span v-if="item.status == '0'" style="margin-top:10px;" class="el-icon el-icon-status-MME redIcon"></span> 
                                        <div class="mme-info">
                                            <span>MME IP : {{item.mmeIp}}</span>
                                            <span v-if="item.status==='0'"><%=rb.getString("MMEZhuangTai")%> : <%= rb.getString("MMEWeiLianJie")%></span>
                                            <span v-if="item.status=='1'"><%=rb.getString("MMEZhuangTai")%> : <%= rb.getString("MMEYiLianJie")%></span>
                                            <span v-if="item.status=='2'"><%=rb.getString("MMEZhuangTai")%> : --</span>
                                            <span v-if="item.status==='' || item.status===null"><%=rb.getString("MMEZhuangTai")%> : </span>
                                        </div>
                                    </div>
                                </div>
                            </el-popover>
						</div>
						<div v-if="!['','NULL','null',null,undefined].includes(infoForm.NEW_MME_STATUS)" style="display: flex;align-items: center;">
							<span v-html="mmesStatusFmt(infoForm.NEW_MME_STATUS)" ></span>
							<el-popover title="All MME" popper-class="mmePopoverClass" :append-to-body="false">
                                <span style="color:#4d84ff;cursor:pointer;" slot="reference">
                                    [ {{parseMMEOnNum(infoForm.NEW_MME_STATUS)}}/{{parseMME(infoForm.NEW_MME_STATUS).length}} ]
                                </span>
                                <div class="mme-list">
                                    <i class="el-icon el-icon-close" style="position: absolute; top: -30px; right: 0px;" onclick="document.body.click()"></i>
                                    <div class="mme-list-item" v-for="item in parseMME(infoForm.NEW_MME_STATUS)">
                                        <span v-if="item.status == '1'" class="el-icon el-icon-status-MME greenIcon"></span>
                                        <span v-if="item.status == '0'" class="el-icon el-icon-status-MME redIcon"></span> 
                                        <div class="mme-info">
                                            <span>MME IP : {{item.mmeIp}}</span>
                                            <span v-if="item.status==='0'"><%=rb.getString("MMEZhuangTai")%> : Disconnected</span>
                                            <span v-if="item.status=='1'"><%=rb.getString("MMEZhuangTai")%> : Connected</span>
                                            <span v-if="item.status=='2'"><%=rb.getString("MMEZhuangTai")%> : --</span>
                                            <span v-if="item.status==='' || item.status===null"><%=rb.getString("MMEZhuangTai")%> : </span>
                                            <span><%=rb.getString("PLMN")%> : {{item.plmnId}}</span>
                                        </div>
                                    </div>
                                </div>
                            </el-popover>
						</div>
					</div>
					<div class="info-item-cls" label="<%=rb.getString("RSMME")%>" v-html="mmeIPfmt(detailForm.mmeStatus, detailForm)"></div>
					<div class="info-item-cls" label="<%=rb.getString("SASKaiGuan")%>" v-show="hasSAS" v-html="sasEnableFmt(detailForm.mmeStatus, detailForm)"></div>
					<div class="info-item-cls" label="<%=rb.getString("SASZhuangTai")%>" v-show="hasSAS" v-html="sasStateFmt(detailForm.sasState, detailForm)"></div>
				</div>
			</div>

			<!-- topo list -->
			<div v-show="topolistShow" class="detail-panel" style="width: 780px;">
				<div class="detail-header">
					<span style="font-weight: bold;color: #7A7992;">
						TOPO
						<span style="color: rgba(0,0,0,.16)"> ({{queryParamsEURU.smallCellCode}}) </span>
					</span>
					<div>
						<i v-show="listOrTopoType == 'list' && ['gNB'].includes(queryParamsEURU.type)" @click="changeListOrTopo('topo')" class="el-icon el-icon-operation-topo"></i>
						<i v-show="listOrTopoType == 'topo' && ['gNB'].includes(queryParamsEURU.type)" @click="changeListOrTopo('list')" class="el-icon el-icon-table"></i>
						<i @click="topolistShow = false" class="el-icon el-icon-close" style="margin-left: 15px;"></i>
					</div>
				</div>
				<div class="detail-body" style="padding: 15px 20px;">
					<!-- list -->
					<div v-show="listOrTopoType == 'list'">
						<p style="display: inline-block;">
							<span class="title-icon" style="vertical-align: sub;"></span> 
							<span style="font-size: 14px; font-weight: bold;">HUB</span>
						</p>
						<el-ctable height="200px" class="border" style="border-radius: 5px;"
							:url="euUrl"
							:query-params="queryParamsEURU"
							:rownumber="false"
						>
							<el-table-column v-if="['eNB'].includes(queryParamsEURU.type)" label="<%=rb.getString("LuYouSuoYin") %>" key="route_index" prop="route_index" width="100"></el-table-column>
							<el-table-column v-if="['eNB'].includes(queryParamsEURU.type)" label="<%=rb.getString("ZhuangTai") %>" key="status" prop="status">
								<template slot-scope="scope">
									<div v-if="scope.row.status == '1'">
										<span class='el-icon el-icon-status-conn-on' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("ZhengChang") %></span>
									</div>
									<div v-if="scope.row.status == '2'">
										<span class='el-icon el-icon-status-conn-off' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("LiXian") %></span>
									</div>
									<div v-if="scope.row.status == '3'">
										<span class='el-icon el-icon-status-alarm' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("GaoJingGuanLi") %></span>
									</div>
									<div v-if="scope.row.status == '4'">
										<span class='el-icon el-icon-status-upgrading' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("ShengJiZhong") %></span>
									</div>
									<div v-if="scope.row.status == '5'">
										<span class='el-icon el-icon-status-upgrade-success' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("ShengJi") %><%=rb.getString("ChengGong") %></span>
									</div>
								</template>
							</el-table-column>
							<el-table-column v-if="['eNB'].includes(queryParamsEURU.type)" label="<%=rb.getString("XiaoZhanBianMa") %>" key="serial_number" prop="serial_number" width="180"></el-table-column>
							<el-table-column v-if="['eNB'].includes(queryParamsEURU.type)" label="<%=rb.getString("MoKuaiXingHao") %>" key="model_name" prop="model_name"></el-table-column>
							<el-table-column v-if="['eNB'].includes(queryParamsEURU.type)" label="<%=rb.getString("RuanJianBanBen") %>" key="software_version" prop="software_version" width="140"></el-table-column>

							<el-table-column v-if="['gNB'].includes(queryParamsEURU.type)" label="<%=rb.getString("LuYouSuoYin") %>" key="RouteIndex" prop="RouteIndex" width="100"></el-table-column>
							<el-table-column v-if="['gNB'].includes(queryParamsEURU.type)" label="<%=rb.getString("ZhuangTai") %>" key="Status" prop="Status">
								<template slot-scope="scope">
									<div v-if="scope.row.Status == '1'">
										<span class='el-icon el-icon-status-conn-on euStatus'></span><span class="statusTip"><%=rb.getString("ZhengChang") %></span>
									</div>
									<div v-if="scope.row.Status == '2'">
										<span class='el-icon el-icon-status-conn-off euStatus'></span><span class="statusTip"><%=rb.getString("LiXian") %></span>
									</div>
									<div v-if="scope.row.Status == '3'">
										<span class='el-icon el-icon-status-alarm euStatus'></span><span class="statusTip"><%=rb.getString("GaoJingGuanLi") %></span>
									</div>							
								</template>
							</el-table-column>
							<el-table-column v-if="['gNB'].includes(queryParamsEURU.type)" label="<%=rb.getString("XiaoZhanBianMa") %>" key="SerialNumber" prop="SerialNumber"></el-table-column>
							<el-table-column v-if="['gNB'].includes(queryParamsEURU.type)" label="<%=rb.getString("MoKuaiXingHao") %>" key="ModelName" prop="ModelName"></el-table-column>
							<el-table-column v-if="['gNB'].includes(queryParamsEURU.type)" label="<%=rb.getString("RuanJianBanBen") %>" key="SoftwareVersion" prop="SoftwareVersion"></el-table-column>
						</el-ctable>

						<p style="display: inline-block; margin-top: 20px;">
							<span class="title-icon" style="vertical-align: sub;"></span> 
							<span style="font-size: 14px; font-weight: bold;">RRU</span>
						</p>
						<el-ctable height="200px" class="border" style="border-radius: 5px;"
							:url="ruUrl"
							:query-params="queryParamsEURU"
							:rownumber="false"
						>
							<el-table-column v-if="['eNB'].includes(queryParamsEURU.type)" label='<%=rb.getString("LuYouSuoYin") %>' key="enb_route_index" prop="route_index" width="100"></el-table-column>
							<el-table-column v-if="['eNB'].includes(queryParamsEURU.type)" label='<%=rb.getString("ZhuangTai") %>' key="enb_status" prop="status">
								<template slot-scope="scope">
									<div v-if="scope.row.status == '1'">
										<span class='el-icon el-icon-status-conn-on' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("ZhengChang") %></span>
									</div>
									<div v-if="scope.row.status == '2'">
										<span class='el-icon el-icon-status-conn-off' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("LiXian") %></span>
									</div>
									<div v-if="scope.row.status == '3'">
										<span class='el-icon el-icon-status-alarm' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("GaoJingGuanLi") %></span>
									</div>
									<div v-if="scope.row.status == '4'">
										<span class='el-icon el-icon-status-upgrading' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("ShengJiZhong") %></span>
									</div>
									<div v-if="scope.row.status == '5'">
										<span class='el-icon el-icon-status-upgrade-success' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("ShengJi") %><%=rb.getString("ChengGong") %></span>
									</div>
								</template>
							</el-table-column>
							<el-table-column v-if="['gNB'].includes(queryParamsEURU.type)" label='<%=rb.getString("LuYouSuoYin") %>' key="gnb_RouteIndex" prop="RouteIndex" width="100"></el-table-column>
							<el-table-column v-if="['gNB'].includes(queryParamsEURU.type)" label='<%=rb.getString("ZhuangTai") %>' key="nb_Status" prop="Status">
								<template slot-scope="scope">
									<div v-if="scope.row.Status == '1'">
										<span class='el-icon el-icon-status-conn-on euStatus'></span><span class="statusTip"><%=rb.getString("ZhengChang") %></span>
									</div>
									<div v-if="scope.row.Status == '2'">
										<span class='el-icon el-icon-status-conn-off euStatus'></span><span class="statusTip"><%=rb.getString("LiXian") %></span>
									</div>
									<div v-if="scope.row.Status == '3'">
										<span class='el-icon el-icon-status-alarm euStatus'></span><span class="statusTip"><%=rb.getString("GaoJingGuanLi") %></span>
									</div>							
								</template>
							</el-table-column>
							<el-table-column v-if="['eNB'].includes(queryParamsEURU.type)" label='<%=rb.getString("XiaoZhanBianMa") %>' key="enb_serial_number" width="120" prop="serial_number"></el-table-column>
							<el-table-column v-if="['gNB'].includes(queryParamsEURU.type)" label='<%=rb.getString("XiaoZhanBianMa") %>' key="gnb_SerialNumber" width="120" prop="SerialNumber"></el-table-column>
							<el-table-column v-if="['gNB'].includes(queryParamsEURU.type)" label="TxGain" prop="TxGain"></el-table-column>
							<el-table-column label='<%=rb.getString("FangSheZhuangTai") %>' width="100" prop="RFTxStatus">
								<template slot-scope="scope">
									<div v-if="['false','0'].includes(scope.row.rf_tx_status) || ['false','0'].includes(scope.row.RFTxStatus)">
										<span class='el-icon el-icon-status-disable' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("GuanBi") %></span>
									</div>
									<div v-if="['true','1'].includes(scope.row.rf_tx_status) || ['true','1'].includes(scope.row.RFTxStatus)">
										<span class='el-icon el-icon-status-enable' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("KaiQi") %></span>
									</div>
								</template>
							</el-table-column>
							<el-table-column v-if="['eNB'].includes(queryParamsEURU.type)" label="<%=rb.getString("MoKuaiXingHao") %>" key="enb_model_name" prop="model_name"></el-table-column>
							<el-table-column v-if="['eNB'].includes(queryParamsEURU.type)" label="<%=rb.getString("RuanJianBanBen") %>" key="enb_software_version" prop="software_version" width="140"></el-table-column>
							<el-table-column v-if="['gNB'].includes(queryParamsEURU.type)" label='<%=rb.getString("MoKuaiXingHao") %>' width="100" key="gnb_ModelName" prop="ModelName"></el-table-column>
							<el-table-column v-if="['gNB'].includes(queryParamsEURU.type)" label='<%=rb.getString("RuanJianBanBen") %>' width="130" key="gnb_SoftwareVersion" prop="SoftwareVersion"></el-table-column>
						</el-ctable>
					</div>
					
					<div v-show="listOrTopoType == 'list' && ['GSM'].includes(queryParamsEURU.type)">
						<p style="display: inline-block;">
							<span class="title-icon" style="vertical-align: sub;"></span> 
							<span style="font-size: 14px; font-weight: bold;">BTS</span>
						</p>
						<el-ctable height="400px" class="border" style="border-radius: 5px;"
							:url="btsUrl"
							:query-params="queryParamsEURU"
							:rownumber="true"
						>
							<el-table-column label="IpaUnitld" prop="IpaUnitld"></el-table-column>
							<el-table-column label="IpOmIRemoteIp" prop="OmIRemoteIp"></el-table-column>
							<el-table-column label="IpOmIRemoteIpBak" prop="OmIRemoteIpBak"></el-table-column>
							<el-table-column label="Status" prop="BscLinkStatus">
								<template slot-scope="scope">
									<div v-if="scope.row.product == 'BTS' || true">
										<span v-if="scope.row.BscLinkStatus == '0'"><%=rb.getString("MMEWeiLianJie")%></span>
										<span v-if="scope.row.BscLinkStatus == '1'"><%=rb.getString("MMEYiLianJie")%></span>
									</div>
									<div v-if="scope.row.product == 'BSC'">--</div>
								</template>
							</el-table-column>
						</el-ctable>
					</div>
					<!-- topo -->
					<div v-show="listOrTopoType == 'topo'">
						<div id="mapSiteTopoPage" style="height: 450px;width: 100%;"></div>
					</div>
				</div>
			</div>
		</div>

		<!-- Modify Site -->
		<el-dialog ref="siteModify" :class="{'pick-model': siteModifyHidden}" style="top: 100px;"
			:modal="false"
			:close-on-click-modal="false"
			:append-to-body="false"
			title='<%=rb.getString("DingWei")%>'
			:width="400"
			:visible.sync="siteModifyShow"
		>
			<el-form ref="modifyForm" :model="siteModifyForm" :rules="modifyRule" label-position="top" size="mini">
				<div v-if="siteModifyForm.siteName" style="font-weight: bold;padding-bottom: 20px;">
					<span><%=rb.getString("ZhanZhiMingCheng")%>: </span>
					<span style="font-weight: normal;word-break: break-all;">{{siteModifyForm.siteName}}</span>
				</div>
				<div v-if="siteModifyForm.serialNumber" style="font-weight: bold;padding-bottom: 20px;">
					<span><%=rb.getString("XiaoZhanBianMa")%>: </span>
					{{siteModifyForm.serialNumber}}
				</div>
				</el-form-item>
				<el-form-item label='<%=rb.getString("WeiDu")%>' style="margin-bottom: 20px;" prop="latitude">
					<el-input v-model="siteModifyForm.latitude"></el-input>
				</el-form-item>
				<el-form-item label='<%=rb.getString("JingDu")%>' prop="longitude">
					<el-input v-model="siteModifyForm.longitude"></el-input>
				</el-form-item>
			</el-form>

			<div slot="footer" style="padding: 0 10px;display: flex;justify-content: space-between;align-items: center;">
				<span @click="modifyPickOnMAP" style="color: #689CF3;cursor: pointer;display: flex;align-items: center;">
					<i class="el-icon el-icon-draw"></i> 
					<span style="text-decoration: underline;margin-left: 5px;">Pick on MAP</span>
				</span>
				<span>
					<el-button type="primary" @click="modifySiteSubmit"><%=rb.getString("QueDing")%></el-button>
					<el-button @click="siteModifyShow = false"><%=rb.getString("QuXiao")%></el-button>
				</span>
			</div>
		</el-dialog>

		<!-- Add Device to Site -->
		<el-dialog ref="siteDevice" style="top: 50px;"
			:modal="false"
			:close-on-click-modal="false"
			:append-to-body="false"
			title='<%=rb.getString("TianJia")%>'
			:width="800"
			:visible.sync="addSiteDeviceShow"
		>
			<el-form ref="siteDeviceFm" :model="deviceToSiteForm" size="small">
				<el-form-item label='<%=rb.getString("TianJiaLeiXing")%>'>
					<el-checkbox-group v-model="deviceToSiteType" @change="deviceToSiteTypeChange" size="small">
						<el-checkbox v-if="isGSMEnable" label="GSM" border></el-checkbox>
						<el-checkbox v-if="enbEnable" label="eNB" border></el-checkbox>
						<el-checkbox v-if="gnbEnable" label="gNB" border></el-checkbox>
					</el-checkbox-group>
				</el-form-item>
				<el-form-item>
					<el-ctable ref="deviceToSiteTb" class="border" style="height: 300px;border-radius: 5px;"
						row-key="small_cell_code"
						:url="addDeviceURL"
						:query-params="deviceToSiteQuery"
						:show-pager="false"
						@selection-change="deviceToSiteSelectChange"
					>
						<template slot="toolbar">
							<div style="display: flex;align-items: center;">
								<span style="font-weight: bold;font-size: 14px;"><%=rb.getString("KPISheBei")%></span>
								<el-query @query="deviceToSiteQueryClick" type="normal" placeholder="SN/Cell Name"></el-query>
							</div>
						</template>
						<el-table-column type="selection" :reserve-selection="true"></el-table-column>
						<el-table-column width="35">
							<template slot-scope="scope">
								<div style="zoom: 0.98;" :class="{
									'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
									'':scope.row.have_connected==2,
									'conn_exc':scope.row.connection_status=='Exception',
									'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
							</template>
						</el-table-column>
						<el-table-column label='<%=rb.getString("XiaoZhanBianMa") %>' prop="serial_number"></el-table-column>
						<el-table-column label='<%=rb.getString("HostName")%>' prop="host_name"></el-table-column>
						<el-table-column label='<%=rb.getString("Type")%>' prop="stationType" width="50"></el-table-column>
					</el-ctable>
					
					<el-checkbox style="margin-top: 10px;"
						v-model="deviceToSiteForm.locationConsistent"
						:true-label="true"
						:false-label="false"
					>
						After checking, the longitude and latitude of the base station will be consistent with the site.
					</el-checkbox>
				</el-form-item>
			</el-form>
			
			<div slot="footer" style="padding-left: 10px;">
				<el-button @click="addSiteDeviceSubmit" type="primary"><%=rb.getString("QueDing")%></el-button>
				<el-button @click="addSiteDeviceShow = false"><%=rb.getString("QuXiao")%></el-button>
			</div>
		</el-dialog>

		<el-dialog ref="gpsdg" title="<%=rb.getString("WeiZhi")%>" :visible.sync="gpsdgVisible" :width="400" :append-to-body="false">
			<el-form ref="gpsform" :model="gpsForm" :rules="gpsRules" label-position="top">
				<el-form-item v-if="activeName == 'ENB'">
					SN<%=rb.getString("MaoHao")%> {{curSerialNumber}}
				</el-form-item>
				<el-form-item v-else>
					<%=rb.getString("MACDiZhi")%><%=rb.getString("MaoHao")%> {{curMacAddress}}
				</el-form-item>
				
				<el-form-item label="<%=rb.getString("WeiDu")%><%=rb.getString("MaoHao")%>" prop="lat">
					<el-input v-model="gpsForm.lat"></el-input>
				</el-form-item>
				<el-form-item label="<%=rb.getString("JingDu")%><%=rb.getString("MaoHao")%>" prop="lon">
					<el-input v-model="gpsForm.lon"></el-input>
				</el-form-item>
				<el-form-item label="<%=rb.getString("GaoDu")%><%=rb.getString("MaoHao")%>" prop="height">
					<el-input v-model="gpsForm.height"></el-input>
				</el-form-item>
				
				<el-form-item v-if="gpsDeviceType == 'enb'" prop="mechanical_downtilt" label="<%=rb.getString("JiXieXiaQingJiao")%>">
					<el-input v-model="gpsForm.mechanical_downtilt" maxLength="5" placeholder="Range: 0-9"><el-input>
				</el-form-item>
				<el-form-item v-if="gpsDeviceType == 'enb'" prop="vertical_3dB_beam_width" label="<%=rb.getString("ChuiZhiBoSuKuanDu")%>">
					<el-input v-model="gpsForm.vertical_3dB_beam_width" maxLength="5" placeholder="Range: 1-9"><el-input>
				</el-form-item>
				<el-form-item v-if="gpsDeviceType == 'enb'" prop="horizontal_azimuth" label="<%=rb.getString("ShuiPinFangWeiJiao")%>">
					<el-input v-model="gpsForm.horizontal_azimuth" maxLength="5" placeholder="Range: 0-359"><el-input>
				</el-form-item>
			</el-form>
			<div>
				<el-button type="primary" @click="setGPS"><%=rb.getString("QueDing")%></el-button>
				<el-button @click="gpsdgVisible = false"><%=rb.getString("QuXiao")%></el-button>
			</div>
		</el-dialog>

		<el-dialog ref="gpsmove" :width="400" 
			:append-to-body="false"
			:visible.sync='gpsmoveVisible' 
			title='<%=rb.getString("QueRen")%>'
			@close="cancelMove">
			<div>
				<%=rb.getString("ChongSheGPS")%><br> 
				<%=rb.getString("XinJingWeiDu")%>: {{gpsmvoeForm.lat}}, {{gpsmvoeForm.lon}}
			</div>
			<div style="padding-top: 10px;">
				<el-button type="primary" @click="confirmMove"><%=rb.getString("QueDing")%></el-button>
				<el-button @click="cancelMove"><%=rb.getString("QuXiao")%></el-button>
			</div>
		</el-dialog>

		<el-slide ref="uetrace"
			class="small-header"
			:title="ueSlideTitle"
			:url="uetraceURL"
			:footer="false"
			@cancel="closeUETrace"
		></el-slide>

		
		<!-- 围栏配置侧边浮层 -->
		<transition name="slide-fade">
			<div v-if="fenceConfigVisible" class="fence-config-panel">
				<!-- 头部 -->
				<div class="fence-panel-header">
					<span class="fence-panel-title">
						{{opType == 'add'?'<%=rb.getString("TianJia")%>':'<%=rb.getString("XiuGai")%>'}}
					</span>
					<i class="el-icon-close" @click="cancelFenceConfig" style="cursor: pointer; font-size: 18px;"></i>
				</div>
				
				<!-- 内容区域 -->
				<div class="fence-panel-content">
					<el-form :model="currentFenceConfig" label-width="100px" label-position="top" size="small">
						<!-- Enable Switch -->
						<el-form-item>
							<span class="el-form-item__label" style="margin-right: 15px;margin-bottom: 0;"><%=rb.getString("QiYong")%></span>
							<el-switch 
								v-model="currentFenceConfig.enable"
								active-color="#13ce66"
								inactive-color="#ff4949">
							</el-switch>
						</el-form-item>
						
						<!-- Fence Name -->
						<el-form-item label="<%=rb.getString("DianZiWeiLanMingCheng")%>" required>
							<el-input style="width: 100%;" maxlength="50"
								v-model="currentFenceConfig.fenceName" 
								placeholder="<%=rb.getString("DianZiWeiLanMingCheng")%>">
							</el-input>
						</el-form-item>

						<el-form-item v-if="opType == 'edit'" label="<%=rb.getString("WeiLanChongHui")%>">
							<el-button type="primary" size="mini" style="min-width: 35px;padding: 2px;" @click="redrawFence">
								<i class="el-icon el-icon-draw"></i>
							</el-button>
						</el-form-item>
						
						<!-- Device List -->
						<el-form-item label="<%=rb.getString("SheBeiLieBiaoBiaoTi")%>" style="margin-bottom: 0px;"></el-form-item>
						
						<div class="device-list-container">
							<el-ctable 
								:data="currentFenceConfig.deviceList"
								:filterstr="deviceQueryStr"
								:match-keys="['code','cellName']"
								:front-pagination="true"
								:pagination="true"
								stripe
								height="100%">
								<template slot="toolbar">
									<div style="display: flex;align-items: center;" class="input-auto-fit">
										<el-input style="width: 300px;"
											v-model="deviceQueryStr" 
											placeholder='<%=rb.getString("XiaoZhanBianMa")%> / <%=rb.getString("HostName")%>'
											clearable
											suffix-icon="el-icon-search">
										</el-input>
										<div class="batch-operate-btn" style="margin-right: 15px;">
											<i class="el-icon el-icon-batchInput" style="font-size: 14px; padding-right: 5px;"></i> 
											<span @click="showBatchDL"><%=rb.getString("PiLiangShuRu")%></span>
										</div>
									</div>
								</template>
								<el-table-column 
									prop="status"
									width="50"
									align="center">
									<template slot-scope="scope">
										<div :class="{
											'el-icon el-icon-status-conn-off':scope.row.status!='Exception' && scope.row.status!='On' && scope.row.status!='updating' && scope.row.status!=1,
											'conn_exc':scope.row.status=='Exception',
											'el-icon el-icon-status-conn-on':scope.row.status=='on'||scope.row.status=='On'||scope.row.status=='updating'||scope.row.status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.status) }" 
											style='font-size:22px;'>
										</div>
									</template>
								</el-table-column>
								<el-table-column 
									prop="code" 
									label="<%=rb.getString("XiaoZhanBianMa")%>" 
									min-width="140"
									show-overflow-tooltip>
								</el-table-column>
								<el-table-column 
									prop="cellName" 
									label="<%=rb.getString("HostName")%>" 
									min-width="120"
									show-overflow-tooltip>
									<template slot-scope="scope">
										{{ scope.row.cellName }}
										<span class="fence-device-bt">
											<i class="el-icon circleIcon el-icon-circle-close" 
												style="color: #FF4949; cursor: pointer; font-size: 15px;line-height: 26px;background-color: silver;zoom: 0.6;" 
												@click="removeDeviceFromFence(scope.row)"></i>
										</span>
									</template>
								</el-table-column>
							</el-ctable>
						</div>
					</el-form>
				</div>
				
				<!-- 底部按钮 -->
				<div class="fence-panel-footer">
					<el-button size="mini" type="primary" @click="confirmFenceConfig"><%=rb.getString("QueDing")%></el-button>
					<el-button size="mini" @click="cancelFenceConfig"><%=rb.getString("QuXiao")%></el-button>
				</div>
			</div>
		</transition>

		<!-- 查看围栏设备侧边浮层 -->
		<transition name="slide-fade">
			<div v-if="fenceViewVisible" class="fence-config-panel">
				<!-- 头部 -->
				<div class="fence-panel-header">
					<span class="fence-panel-title" style="width: 85%;text-overflow: ellipsis;display: inline;overflow: hidden;"
						:title="fenceDeviceQuery.fenceName"
					>
						{{ fenceDeviceQuery.fenceName }}
					</span>
					<span>
						<i class="el-icon el-icon-operation-export" @click="exportFenceDevices" style="margin-right: 10px;"></i>
						<i class="el-icon-close" @click="closeFenceView" style="cursor: pointer; font-size: 18px;"></i>
					</span>
				</div>
				
				<!-- 内容区域 -->
				<div class="fence-panel-content">
					<el-form label-width="100px" label-position="top" size="small" style="height: 100%;display: flex; flex-direction: column;">
						<!-- 搜索框 -->
						<el-form-item>
							<el-input 
								v-model="fenceDeviceQuery.searchText" 
								placeholder='<%=rb.getString("XiaoZhanBianMa")%> / <%=rb.getString("HostName")%>'
								clearable
								suffix-icon="el-icon-search">
							</el-input>
						</el-form-item>
						
						<!-- Device List -->
						<el-form-item label="Device List" style="margin-bottom: 0px;"></el-form-item>
						
						<div class="device-list-container" style="height: 100%;">
							<el-ctable 
								:data="fenceDeviceList"
								height="100%"
								:filterstr="fenceDeviceQuery.searchText"
								:match-keys="['serialNumber','hostName']"
								:front-pagination="true"
								:pagination="true"
								stripe>
								<el-table-column 
									prop="status"
									width="50"
									align="center">
									<template slot-scope="scope">
										<div :class="{
											'el-icon el-icon-status-conn-off':scope.row.status!='Exception' && scope.row.status!='On' && scope.row.status!='updating' && scope.row.status!=1,
											'conn_exc':scope.row.status=='Exception',
											'el-icon el-icon-status-conn-on':scope.row.status=='on'||scope.row.status=='On'||scope.row.status=='updating'||scope.row.status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.status) }" 
											style='font-size:22px;'>
										</div>
									</template>
								</el-table-column>
								<el-table-column 
									prop="serialNumber" 
									label="<%=rb.getString("XiaoZhanBianMa")%>" 
									min-width="140"
									show-overflow-tooltip>
								</el-table-column>
								<el-table-column 
									prop="hostName" 
									label="<%=rb.getString("HostName")%>" 
									min-width="120"
									show-overflow-tooltip>
								</el-table-column>
								<el-table-column v-if="false"
									prop="type" 
									label="Type" 
									width="70"
									align="center">
									<template slot-scope="scope">
										<el-tag size="mini" :type="getDeviceTypeColor(scope.row.neType)">
											{{ scope.row.neType }}
										</el-tag>
									</template>
								</el-table-column>
							</el-ctable>
						</div>
					</el-form>
				</div>
			</div>
		</transition>
		<!-- 批量输入dialog -->
		<el-dialog class='dialogStyle' title='<%=rb.getString("PiLiangShuRu")%>' width='630px' :visible.sync='batchSnDialog' :append-to-body="true" :close-on-click-modal="false" @close='closeBatchSn'>
			<el-form ref='batchSnForm' :rules='batchSnRules' :model='batchSnForm' label-position="top">
				<div>
					<label><%=rb.getString("XiaoZhanBianMa")%></label>
					<el-form-item prop='serialNumber' style="margin-bottom:22px;">
						<el-input v-model='batchSnForm.serialNumber' type='textarea' :rows="4" style='margin-top:5px;'></el-input>
					</el-form-item>
					<p style='display:flex;color:#BBB'><span class='el-icon el-icon-circle-info' style='font-size:14px;'></span><span style="font-size:12px;"><%=rb.getString("eNBZhuCeTiShiWenZi") %></span></p>
				</div>
				<div style='margin-top:45px;'>
					<el-button @click='saveBatchSn' type="primary"><%=rb.getString("QueDing")%></el-button>
					<el-button @click='closeBatchSn'><%=rb.getString("QuXiao")%></el-button>
				</div>
			</el-form>
		</el-dialog>
		<!-- 批量导入不正确SN dialog，展示失败SN，支持SN内容下载 -->
		<el-dialog class='dialogStyle' width='630px' :visible.sync='batchSnErrorDialog' :append-to-body="true" :close-on-click-modal="false" @close='closeBatchSnError'>
			<div slot="title" style="font-weight: bold; font-size: 16px; padding-top: 8px;">
				<i class="el-icon el-icon-status-alarm" style="color: red;"></i>
				Input Failed
			</div>
			<el-form ref='batchSnErrorForm' :model='batchSnErrorForm' label-position="top">
				<div>
					<label><%=rb.getString("WeiLanPiLiangTiShi")%></label>
					<el-form-item>
						<el-input v-model='batchSnErrorForm.errorSerialNumber' type='textarea' :rows="8" style='margin-top:5px;' readonly></el-input>
					</el-form-item>
				</div>
				<div style='margin-top:45px;'>
					<el-button @click='downloadErrorSns' type="primary">
						<i class="el-icon el-icon-common-download" style="margin-right: 5px;"></i>
						Download the failed file
					</el-button>
					<el-button @click='closeBatchSnError'><%=rb.getString("QuXiao")%></el-button>
				</div>
			</el-form>
		</el-dialog>

		<!-- KPI控制面板 -->
		<transition name="el-fade-in">
			<div v-if="kpiPanelShow" class="kpi-panel">
				<!-- 面板头部 -->
				<div class="kpi-panel-header">
					<span class="kpi-panel-title">KPI</span>
					<span class="kpi-panel-close" @click="closeKpiPanel">×</span>
				</div>
				
				<!-- Display选项 -->
				<div class="kpi-panel-item">
					<div class="kpi-panel-label"><%=rb.getString("KPIXianShi")%></div>
					<el-select v-model="kpiDisplay" size="mini" @change="onKpiDisplayChange" style="width: 60%;zoom: 0.8;">
						<el-option label="UE Count" value="ueCount"></el-option>
						<el-option label="Throughput DL" value="throughputDL"></el-option>
						<el-option label="Throughput UL" value="throughputUL"></el-option>
					</el-select>
				</div>
				
			<!-- Level等级条 -->
			<div class="kpi-panel-item">
				<div class="kpi-panel-label">
					<%=rb.getString("KPIDengJi")%>
					<span v-if="kpiDisplay != 'ueCount'" style="font-size: 10px;">(Mbps)</span>
				</div>
				<div class="kpi-level-bar">
					<div v-for="level in displayKpiLevels" 
						:key="level.level"
						:class="'kpi-level-segment level-' + level.level"
						:title="'Level ' + level.level + ': ' + level.range[0] + '-' + level.range[1]">
						<span class="kpi-level-text">{{level.label}}</span>
					</div>
				</div>
			</div>				<!-- Date日期选择 -->
				<div class="kpi-panel-item">
					<div class="kpi-panel-label"><%=rb.getString("RiQi")%></div>
					<div class="kpi-date-list">
						<div v-for="date in kpiDates" 
							:key="date"
							:class="['kpi-date-item', {active: kpiSelectedDate === date}]"
							@click="selectKpiDate(date)">
							{{formatDateShort(date)}}
						</div>
					</div>
			</div>
			
			<!-- Time时间滑块 -->
			<div class="kpi-panel-item">
				<div class="kpi-panel-label" style="min-width: 60px;"><%=rb.getString("ShiJian")%></div>
				<div class="kpi-time-slider" style="position: relative;">
					<el-slider 
						v-model="kpiSelectedTime" 
						:min="1" 
						:max="24"
						:step="1"
						:show-stops="true"
						:marks="{
							1: '1', 2: '2', 3: '3', 4: '4', 5: '5', 6: '6',
							7: '7', 8: '8', 9: '9', 10: '10', 11: '11', 12: '12',
							13: '13', 14: '14', 15: '15', 16: '16', 17: '17', 18: '18',
							19: '19', 20: '20', 21: '21', 22: '22', 23: '23', 24: '24'
						}"
						@change="onKpiTimeChange">
					</el-slider>
					<!-- 显示当前选中的时间值 -->
					<div class="kpi-slider-value" 
						:style="{left: 'calc(15px + ((100% - 30px) * ' + ((kpiSelectedTime - 1) / 23) + '))'}">
						{{kpiSelectedTime < 10 ? '0' + kpiSelectedTime : kpiSelectedTime}}:00
					</div>
				</div>
			</div>
				<!-- Display value复选框 -->
				<div class="kpi-panel-item">
					<el-checkbox v-model="kpiShowValue" @change="onKpiShowValueChange">
						<%=rb.getString("KPIXianShiZhi")%>
					</el-checkbox>
				</div>
			</div>
		</transition>

		<!-- 设置页面滑动面板 -->
		<el-slide ref="topoSettingSlide" class="no-padding commonSlideClass commonWarp" 
            :title="slideTitle" 
			:url="slideURL"
			:position="slidePosition"
			:width="sliderWidth"
			:height="sliderHeight"
			:header='false' :footer="false" 
			@cancel="hideTopoSettingSlide"
		></el-slide>
	</div>
	
<div>
<script type="text/javascript" src="${ctx}/js/packages/topo/topo-helper.js"></script>

<script type="text/javascript">
	var topoOverViewEnable = scenarioKey == 'S00013';
	//RSRP
	var lowVal = localStorage.getItem("rsrp1"),
		highVal = localStorage.getItem("rsrp2");
	
	var globMap, globSiteMap, groupName = '';
    
	var topovm = new Vue({
		el: '#topo_container', 
		data() {
			var vm = this,
				/**
				* 纬度校验
				* @param rule{object}：校验配置的规则
				* @param value{string}: 纬度值
				* @param cb{function}：回调方法
				**/
				latValid = function(rule,value,cb){
					value = value + '';

					if(value) {
						if(isNaN(value)) {
							cb('<%=rb.getString("WeiDuFanWei")%>: [-90,90], <%=rb.getString("JingQueDu")%>: 6');
						}else if(value<-90 || value>90){
							cb('<%=rb.getString("WeiDuFanWei")%>: [-90,90], <%=rb.getString("JingQueDu")%>: 6');
						}else {
							var arr = value.split('.'),
								precision = arr[1]||'';
							if(precision.length>6) {
								cb('<%=rb.getString("WeiDuFanWei")%>: [-90,90], <%=rb.getString("JingQueDu")%>: 6');
							}else cb();
						}
					}else {
						if(rule.required == true) {
							cb('Required');
						}else {
							cb();
						}
					}
				},
				/**
				* 经度校验
				* @param rule{object}：校验配置的规则
				* @param value{string}: 经度值
				* @param cb{function}：回调方法
				**/
				lonValid = function(rule,value,cb){
					value = value + '';

					if(value) {
						if(isNaN(value)) {
							cb('<%=rb.getString("JingDuFanWei")%>: [-180,180], <%=rb.getString("JingQueDu")%>: 6');
						}else if(value<-180 || value>180){
							cb('<%=rb.getString("JingDuFanWei")%>: [-180,180], <%=rb.getString("JingQueDu")%>: 6');
						}else {
							var arr = value.split('.'),
								precision = arr[1]||'';
							if(precision.length>6) {
								cb('<%=rb.getString("JingDuFanWei")%>: [-180,180], <%=rb.getString("JingQueDu")%>: 6');
							}else cb();
						}
					}else {
						if(rule.required == true) {
							cb('Required');
						}else {
							cb();
						}
					}
				},// 高度 表单验证规则
				heightValidator = function(rule,value,cb) {
					if(value) {
						if(value<=99999999 && value-0>=0) {
							cb();
						}else {
							cb('<%=rb.getString("QuZhiFanWei")%>: 0-99999999');
						}
					}else {
						cb();
					}
				},
				validSiteName = (rule,value,cb) => {
					var trimVal = value.trim();

					if(trimVal === '') {
						cb('Required');
					}else {
						cb();
					}
				},
				validMechanicalDowntilt = function(rule,value,callback) {
					var msg = 'Range: 0-9';
					
					if(['', null, undefined].includes(value)) {
						callback();
					}else {
						if(isNaN(value) || value - 0 < 0 || value - 9 > 0) {
							callback(msg)
						}else {
							callback();
						}
					}
				},
				validVertical3dB = function(rule,value,callback) {
					var msg = 'Range: 1-9';
					
					if(['', null, undefined].includes(value)) {
						callback();
					}else {
						if(isNaN(value) || value - 1 < 0 || value - 9 > 0) {
							callback(msg)
						}else {
							callback();
						}
					}
				},
				validHorizontalAzim = function(rule,value,callback) {
					var msg = 'Range: 0-359';
					
					if(['', null, undefined].includes(value)) {
						callback();
					}else {
						if(isNaN(value) || value - 0 < 0 || value - 359 > 0) {
							callback(msg)
						}else {
							callback();
						}
					}
				}
				validatorSn = (rule,value,callback) => {
					var serialNumber = value, temp = /^(\d|[a-zA-Z]|-|\s){1,30}$/,
						list = serialNumber.split(/[\s,;]+/).map(sn => sn.trim()).filter(sn => sn !== '');
			
					if(list.length > 0){
						var nameFlag = list.every(function(item,index){
							return temp.test(item)
						})
						if(nameFlag){
							callback()
						}else{
							callback(new Error('<%=rb.getString("QingShuRuZhengQueSn")%>'));
						}
						
					}else if (serialNumber == null || serialNumber.length == 0) {
						callback(new Error('<%=rb.getString("SNBuNengWeiKong")%>'));
					}else{
						callback();
					}
				};
			var siteDefaultCk = [],
				siteLowCk = [];
			if(writableMap['CODE_ENB_MONITOR'] != undefined) {
				siteDefaultCk.push('eNB');
				siteLowCk.push('enb');
			}
			if(writableMap['CODE_GNB_MONITOR'] != undefined) {
				siteDefaultCk.push('gNB');
				siteLowCk.push('gnb');
			}
			if(supportGSM == true) {
				siteDefaultCk.push('GSM');
				siteLowCk.push('gsm');
			}
			
			return {
				fenceQueryStr: '',
				isFenceEnable: isFenceEnable == true,
				batchSnDialog: false,
				batchSnForm:{
					serialNumber: '',
					type: 'input'
				},
				batchSnRules:{
					serialNumber:[
						{validator: validatorSn,trigger:'change'}
					]
				},
				batchSnErrorDialog: false,
				batchSnErrorForm:{
					errorSerialNumber: ''
				},

				fenceEnable: false,
				opType: '',

				uetraceURL: '',
				ueSlideTitle: '',

				// 设置页面滑动面板相关属性
				slideURL: '',
                slideTitle: '',
				slidePosition: 'top',
				sliderHeight: '100%',
				sliderWidth: '72%',

				topoOverViewEnable: topoOverViewEnable,
				gpsDeviceType: '',
				activeType: 'group',
				selectedCode: '',

				groupIds: '',
				limit: 100, // 过滤数据: 阀值
				links: {}, // enb为key，cpe存于数组中：enbCode:[cpeCode1, cpeCode2, ...]，建立 1:n 关系
				activeName: 'ENB',
				groupURL: '${ctx}/system/deviceGroup/getDeviceGroupList.action',
				deviceURL: '',
				cpeDeviceURL: '',
				// 地图初始化标记
				isMapInitialized: false, // 标记地图是否已完成首次初始化
				savedMapView: null, // 保存的地图视图状态（中心点和缩放级别）
				// 设备组查询参数
				groupQueryParams: {
					search_text: ''
				},
				// enb列表查询参数
				enbQueryParams: {
					device_group_id: '',
					device_type: 'ENB',
					search_text: '',
					operator_code: operator_code,
					no_gps: ''
				},
				gnbQueryParams: {
					device_group_id: '',
					device_type: 'GNB',
					search_text: '',
					operator_code: operator_code,
					no_gps: ''
				},
				gsmQueryParams: {
					device_group_id: '',
					device_type: 'GSM',
					search_text: '',
					operator_code: operator_code,
					no_gps: ''
				},
				// cpe列表查询参数
				cpeQueryParams: {
					device_group_id: '',
					device_type: 'CPE',
					search_text: '',
					operator_code: operator_code,
					no_gps: ''
				},
				// 显示状态表单 -- 提交用
				form: {
					deviceType: ['enb'],
					deviceStatus: ['on','off'],
					activeStatus: [],
					sasStatus: [],
					ueStatus: [],
					nameStatus: []
				},
				// 显示状态表单 -- 界面展示用
				statusForm: {
					deviceType: ['enb'],
					deviceStatus: ['on','off'],
					activeStatus: [],
					sasStatus: [],
					ueStatus: [],
					nameStatus: []
				},
				statusType: 'deviceStatus',
				// 搜索和重叠节点面板相关
				searchPanelShow: false,
				searchKeyword: '',
				searchResults: [],
				clusterPanelShow: false,
				clusterNodes: [],
				// KPI控制面板相关
				kpiPanelShow: false,
				kpiMode: false, // KPI独立渲染模式
				kpiDisplay: 'ueCount', // ueCount | throughputDL | throughputUL
				kpiSelectedDate: '', // 选中的日期
				kpiDates: [], // 近7天日期列表
				kpiSelectedTime: new Date(gloableTime).getHours() || 24, // 选中的时刻（1-24），默认为当前小时
				kpiShowValue: false, // 是否显示节点值
				// UE Count 使用 5 个等级
				kpiLevels5: [],
				kpiLevelsDL: [],
				kpiLevelsUL: [
					{ level: 1, color: '#FF5B45', range: [0, 5], label: '<5' },
					{ level: 2, color: '#FF8F1F', range: [5, 10], label: '5-10' },
					{ level: 3, color: '#FFD21F', range: [10, 15], label: '10-15' },
					{ level: 4, color: '#11B067', range: [15, 20], label: '15-20' },
					{ level: 5, color: '#0371D7', range: [20, 25], label: '20-25' },
					{ level: 6, color: '#7736A7', range: [25, 10000], label: '≥25' }
				],
				// Throughput DL/UL 使用 7 个等级
				kpiLevels7: [
					{ level: 1, color: '#FF5B45', range: [0, 25], label: '<25' },
					{ level: 2, color: '#FF8F1F', range: [25, 100], label: '25-100' },
					{ level: 3, color: '#FFD21F', range: [100, 300], label: '100-300' },
					{ level: 4, color: '#11B067', range: [300, 500], label: '300-500' },
					{ level: 5, color: '#0371D7', range: [500, 1000], label: '500-1G' },
					{ level: 6, color: '#7736A7', range: [1000, 2000], label: '1G-2G' },
					{ level: 7, color: '#861786', range: [2000, 10000000], label: '≥2G' }
				],
				_savedRenderState: null, // 保存KPI模式前的渲染状态
				// gps设置表单
				gpsForm: {
					lat: '', // 纬度
					lon: '',
					height: '',
					mechanical_downtilt: '',
					vertical_3dB_beam_width: '',
					horizontal_azimuth: '',
				},
				gpsmvoeForm: {
					lat: '', // 纬度
					lon: '', // 经度
					olatlng: '',
					cell_code: '',
					operator_code: operator_code,
					target: '',
				},
				// gps域校验规则
				gpsRules: {
					lat:[
						{validator: latValid}
					],
					lon:[
						{validator: lonValid}
					],
					height: [{validator: heightValidator}],
					mechanical_downtilt: [{validator: validMechanicalDowntilt}],
					vertical_3dB_beam_width: [{validator: validVertical3dB}],
					horizontal_azimuth: [{validator: validHorizontalAzim}]
				},
				infoForm: {
					cellCode: '',
					active: '',
					alarm: {},
					code: '',
					groupName: '',
					ip: '',
					lat: '',
					lon: '',
					mmeEnable: '',
					mmePool1: '',
					mmePool2: '',
					mmeStatus: '',
					name: '',
					online: '',
					serverList: '',
					sasEnable: '',
					sasState: '',
					sn: '',
					NEW_MME_STATUS: '',
					RSRP0: '',
					RSRP1: '',
					pci: '',
					mechanical_downtilt: '',
					vertical_3dB_beam_width: '',
					horizontal_azimuth: '',
					minRadius: '',
					radius: '',
					ueCount: ''
				},
				cpeNodes: [], // 记录获取的cpe节点
				enbNodes: [], // 记录获取的enb节点
				gnbNodes: [], // 记录获取的gnb节点
				gsmNodes: [], // 记录获取的gsm节点
				prevNodes: [], // 缓存的上次绘制节点
				gpsdgVisible: false,
				gpsmoveVisible: false,
				allNodes: [], // cpe、enb节点合集
				curDeviceCode: '',
				curSerialNumber: '',
				curMacAddress: '',
				sasEnable: true,
				groupShow: false,
				curRow: {},
				groupNames: '',
				firtLoad: true,
				infoShow: false,

				siteInfoShow: false,
				siteMenus: [],
				siteListURL: '',
				siteQuery: {
					siteName: '',
					rd: ''
				},
				siteType: '',
				currentSiteRow: {},
				addDeviceURL: '${ctx}/site/getStationList.action',
				viewDeviceURL: '',
				siteDefaultCk: siteDefaultCk,
				siteAddDeviceType: siteDefaultCk,
				deviceToSiteType: siteDefaultCk,
				addQuery: {
					deviceType: siteDefaultCk.join(','),
					searchText: '',
					page: 1,
					rows: 20
				},
				deviceToSiteQuery: {
					deviceType: siteDefaultCk.join(','),
					searchText: '',
					page: 1,
					rows: 20
				},
				viewQuery: {
					queryType: 'groupid',
					device_type: siteDefaultCk.join(','),
					operator_code: operator_code,
					siteName: '',
					search_text: '',
					rd: ''
				},
				siteAddForm: {
					siteName: '',
					latitude: '',
					longitude: '',
					cellCodes: '',
					locationConsistent: false,
				},
				deviceToSiteForm: {
					siteName: '',
					cellCodes: '',
					locationConsistent: false,
				},
				addRule: {
					siteName: [{validator: validSiteName}],
					latitude: [{validator: latValid, required: true}],
					longitude: [{validator: lonValid, required: true}]
				},
				siteModifyForm: {
					serialNumber: '',
					cellCode: '',
					siteName: '',
					latitude: '',
					longitude: '',
				},
				modifyRule: {
					latitude: [{validator: latValid, required: true}],
					longitude: [{validator: lonValid, required: true}],
				},
				siteModifyShow: false,
				siteModifyHidden: false,
				deviceDetailShow: false,
				siteAddHidden: false,
				
				// OpenLayers 事件处理器引用（用于移除监听）
				modifyClickHandler: null,
				addPickClickHandler: null,
				
				detailForm: {
					cellCode: '',
					active: '',
					alarm: {},
					code: '',
					groupName: '',
					ip: '',
					lat: '',
					lon: '',
					mmeEnable: '',
					mmePool1: '',
					mmePool2: '',
					mmeStatus: '',
					name: '',
					online: '',
					serverList: '',
					sasEnable: '',
					sasState: '',
					sn: '',
					NEW_MME_STATUS: '',
					RSRP0: '',
					RSRP1: ''
				},
				addSiteDeviceShow: false,
				topolistShow: false,
				allSiteLatlonNodes: [],
				listOrTopoType: 'list',
				dataURL: '',
				euUrl: '',
				ruUrl: '',
				btsUrl: '',
				queryParamsEURU:{
					smallCellCode: '',
					type: ''
				},
				
				siteEnable: supportTopoSite,
				siteDeviceType: siteLowCk,
				siteCheckAll: siteLowCk.length? true:false,
				isIndeterminate: false,
				isGSMEnable: supportGSM == true,

				overViewList: [],
				overviewCheck: 'false',

				siteTab: 'deviceList',
				siteNameShow: true,
				
				// 电子围栏相关数据
				fenceList: [], // 存储所有围栏参数
				fenceConfigVisible: false, // 围栏配置面板显示状态
				deviceQueryStr: '', 
				currentFenceConfig: {
					enable: true,
					fenceName: '',
					deviceList: [], // 圈中的设备列表
					fenceParams: null // 当前围栏的几何参数
				},
				// 查看围栏设备浮层
				fenceViewVisible: false, // 查看设备面板显示状态
				fenceDeviceList: [], // 存储围栏内设备列表
				fenceDeviceQuery: {
					fenceId: '',
					searchText: '',
					fenceName: ''
				}
			};
		},
		computed: {
			// 根据 kpiDisplay 返回对应的等级配置
			displayKpiLevels() {
				if (this.kpiDisplay === 'ueCount') {
					return this.kpiLevels5; // UE Count 使用 5 个等级
				} else if( this.kpiDisplay === 'throughputDL') {
					return this.kpiLevelsDL; // Throughput DL
				} else if( this.kpiDisplay === 'throughputUL') {
					return this.kpiLevelsUL; // Throughput UL
				}else {
					return [];
				}
			},
			enbEnable() {
				return writableMap['CODE_ENB_MONITOR'] != undefined;
			},
			gnbEnable() {
				return writableMap['CODE_GNB_MONITOR'] != undefined;
			},
			cpeEnable() {
				return writableMap['CODE_CPE_MONITOR'] != undefined;
			},
			hasSAS() {
				return writableMap['CODE_ADVANCE_SAS'] != undefined;
			},
			isLocal() {
				return isLocal == 'true';
			},
			infoPanelCls() {
				return {
					'info-panel': true,
					'show': this.infoShow
				};
			},
			sitePanelCls() {
				return {
					'site-panel': true,
					'show': this.siteInfoShow,
					'pick-model': this.siteAddHidden
				};
			},
			// 设备组展开设备列表时 箭头样式
			arrowClass() {
				var vm = this;

				return {
					'group-arrow': true,
					'el-icon': true,
					'el-icon-left':  vm.groupShow,
					'el-icon-right':  !vm.groupShow
				}
			},
			// sas开关状态样式 -- 控制是否可修改gps
			sasSwitchCls() {
				var vm = this;

				return  {
					'sas-switch-on': vm.sasEnable
				}
			},
			deviceInfos() {
				var vm = this,
					allNodes = vm.allNodes.filter(function(item){
						return !['',null].includes(item.lat) && !['',null].includes(item.lon);
					}),
					// 总数统计
					allEnbNodes = vm.allNodes.filter(function(item){
						return !['',null].includes(item.lat) && !['',null].includes(item.lon) && item.type == 'enb' && item.isGSM != true && item.isGnb != true;
					}),
					allGnbNodes = vm.allNodes.filter(function(item){
						return !['',null].includes(item.lat) && !['',null].includes(item.lon) && (item.type == 'gnb' || item.isGnb == true);
					}),
					allCpeNodes = vm.allNodes.filter(function(item){
						return !['',null].includes(item.lat) && !['',null].includes(item.lon) && item.type == 'cpe';
					}),
					allGSMNodes = vm.allNodes.filter(function(item){
						return !['',null].includes(item.lat) && !['',null].includes(item.lon) && (item.type == 'enb' && item.isGSM == true || item.type == 'gsm');
					}),
					// 在线数统计
					enbOnlineList = allNodes.filter(function(item){
						return item.online == 'on' && item.type == 'enb' && item.isGSM != true && item.isGnb != true;
					}),
					gnbOnlineList = allNodes.filter(function(item){
						return item.online == 'on' && (item.type == 'gnb' || item.isGnb == true);
					}),
					cpeOnlineList = allNodes.filter(function(item){
						return item.online == 'on' && item.type == 'cpe';
					}),
					gsmOnlineList = allNodes.filter(function(item){
						return item.online == 'on' && (item.type == 'enb' && item.isGSM == true || item.type == 'gsm');
					}),
					// 激活数统计
					enbActiveList = allNodes.filter(function(item){
						return item.active == 'yes' && item.type == 'enb' && item.isGSM != true && item.isGnb != true;
					}),
					gnbActiveList = allNodes.filter(function(item){
						return item.active == 'yes' && (item.type == 'gnb' || item.isGnb == true);
					}),
					cpeActiveList = allNodes.filter(function(item){
						return item.active == 'yes' && item.type == 'cpe';
					}),
					gsmActiveList = allNodes.filter(function(item){
						return item.active == 'yes' && (item.type == 'enb' && item.isGSM == true || item.type == 'gsm');
					}),
					// ENB SAS状态统计
					enbUnregList = allNodes.filter(function(item){
						return item.sasState == 'Unregistered' && item.type == 'enb' && item.isGSM != true && item.isGnb != true;
					}),
					enbRegList = allNodes.filter(function(item){
						return item.sasState == 'Registered' && item.type == 'enb' && item.isGSM != true && item.isGnb != true;
					}),
					enbGrantedList = allNodes.filter(function(item){
						return item.sasState == 'Granted' && item.type == 'enb' && item.isGSM != true && item.isGnb != true;
					}),
					enbAuthList = allNodes.filter(function(item){
						return item.sasState == 'Authorized' && item.type == 'enb' && item.isGSM != true && item.isGnb != true;
					}),
					// gNB SAS状态统计
					gnbUnregList = allNodes.filter(function(item){
						return item.sasState == 'Unregistered' && (item.type == 'gnb' || item.isGnb == true);
					}),
					gnbRegList = allNodes.filter(function(item){
						return item.sasState == 'Registered' && (item.type == 'gnb' || item.isGnb == true);
					}),
					gnbGrantedList = allNodes.filter(function(item){
						return item.sasState == 'Granted' && (item.type == 'gnb' || item.isGnb == true);
					}),
					gnbAuthList = allNodes.filter(function(item){
						return item.sasState == 'Authorized' && (item.type == 'gnb' || item.isGnb == true);
					}),
					// GSM SAS状态统计
					gsmUnregList = allNodes.filter(function(item){
						return item.sasState == 'Unregistered' && (item.type == 'enb' && item.isGSM == true || item.type == 'gsm');
					}),
					gsmRegList = allNodes.filter(function(item){
						return item.sasState == 'Registered' && (item.type == 'enb' && item.isGSM == true || item.type == 'gsm');
					}),
					gsmGrantedList = allNodes.filter(function(item){
						return item.sasState == 'Granted' && (item.type == 'enb' && item.isGSM == true || item.type == 'gsm');
					}),
					gsmAuthList = allNodes.filter(function(item){
						return item.sasState == 'Authorized' && (item.type == 'enb' && item.isGSM == true || item.type == 'gsm');
					}),
					// CPE SAS状态统计
					cpeUnregList = allNodes.filter(function(item){
						return item.sasState == 'Unregistered' && item.type == 'cpe';
					}),
					cpeRegList = allNodes.filter(function(item){
						return item.sasState == 'Registered' && item.type == 'cpe';
					}),
					cpeGrantedList = allNodes.filter(function(item){
						return item.sasState == 'Granted' && item.type == 'cpe';
					}),
					cpeAuthList = allNodes.filter(function(item){
						return item.sasState == 'Authorized' && item.type == 'cpe';
					}),
					
					
					onlineList = allNodes.filter(function(item){
						return item.online == 'on';
					}),
					activeList = allNodes.filter(function(item){
						return item.active == 'yes';
					}),
					unregList = allNodes.filter(function(item){
						return item.sasState == 'Unregistered';
					}),
					regList = allNodes.filter(function(item){
						return item.sasState == 'Registered';
					}),
					grantedList = allNodes.filter(function(item){
						return item.sasState == 'Granted';
					}),
					authList = allNodes.filter(function(item){
						return item.sasState == 'Authorized';
					});

				return {
					onlineCount: onlineList.length,
					activeCount: activeList.length,
					total: allNodes.length,
					unregistered: unregList.length,
					registered: regList.length,
					granted: grantedList.length,
					authorized: authList.length,
					
					enbTotal: allEnbNodes.length,
					enbOnlineCount: enbOnlineList.length,
					enbActiveCount: enbActiveList.length,
					gnbTotal: allGnbNodes.length,
					gnbOnlineCount: gnbOnlineList.length,
					gnbActiveCount: gnbActiveList.length,
					gsmTotal: allGSMNodes.length,
					gsmOnlineCount: gsmOnlineList.length,
					gsmActiveCount: gsmActiveList.length,
					cpeTotal: allCpeNodes.length,
					cpeOnlineCount: cpeOnlineList.length,
					cpeActiveCount: cpeActiveList.length,

					enbunregistered: enbUnregList.length,
					enbregistered: enbRegList.length,
					enbgranted: enbGrantedList.length,
					enbauthorized: enbAuthList.length,
					gnbunregistered: gnbUnregList.length,
					gnbregistered: gnbRegList.length,
					gnbgranted: gnbGrantedList.length,
					gnbauthorized: gnbAuthList.length,
					gsmunregistered: gsmUnregList.length,
					gsmregistered: gsmRegList.length,
					gsmgranted: gsmGrantedList.length,
					gsmauthorized: gsmAuthList.length,
					cpeunregistered: cpeUnregList.length,
					cperegistered: cpeRegList.length,
					cpegranted: cpeGrantedList.length,
					cpeauthorized: cpeAuthList.length
				};
			},
			siteStatusCount() {
				var vm = this,
					siteNodes = vm.allSiteLatlonNodes.filter(function(item){
						return !['',null].includes(item.lat) && !['',null].includes(item.lon);
					}),
					normalList = siteNodes.filter(function(item){
						return item.status == '1';
					}),
					abnormalList = siteNodes.filter(function(item){
						return item.status == '2';
					}),
					offlineList = siteNodes.filter(function(item){
						return item.status == '3';
					});

				return {
					normalCount: normalList.length,
					abnormalCount: abnormalList.length,
					offlineCount: offlineList.length
				};
			}
		},
		watch: {
			form: {
				deep: true,
				handler(n) {
					var vm = this;
					
					//vm.filterRender(vm.allNodes, globMap, true);
					vm.getTopoInfos(vm.groupIds);
					vm.hidePanel();
				}
			},
			siteAddDeviceType: function(val) {
				var vm = this;

				vm.addQuery.deviceType = val.join(',');
			},
			deviceToSiteType: function(val) {
				var vm = this;

				vm.deviceToSiteQuery.deviceType = val.join(',');
			},
			siteInfoShow: function(val) {
				var vm = this;

				if(val == false) {
					vm.deviceDetailShow = false;
					vm.topolistShow = false;
				}
			}
		},
		methods: {
			// 打开KPI/UE控制面板
			kpiUeClick() {
				var vm = this;
				
				if (!window.globOLHelper) {
					return;
				}
				vm.initKpiDates(); // 初始化近7天日期
				vm.kpiPanelShow = true; // 打开面板
				vm.kpiMode = true; // 进入KPI独立渲染模式
				vm.saveCurrentRenderState(); // 保存当前渲染状态
				vm.getKpiNodeData(); // 刷新节点渲染
				// vm.renderKpiNodes();
			},
			// 初始化KPI日期（近7天）
			initKpiDates() {
				var vm = this;
				var dates = [];
				var today = new Date();
				
				for (var i = 6; i >= 0; i--) {
					var date = new Date(today);
					date.setDate(date.getDate() - i);
					dates.push(vm.formatDate(date));
				}
				
				vm.kpiDates = dates;
				vm.kpiSelectedDate = dates[dates.length - 1]; // 默认选择今天
			},
			// 格式化日期为 MM/DD
			formatDateShort(dateStr) {
				if (!dateStr) return '';
				var parts = dateStr.split('-');
				return parts[1] + '/' + parts[2];
			},
			// 格式化日期为 YYYY-MM-DD
			formatDate(date) {
				var year = date.getFullYear();
				var month = ('0' + (date.getMonth() + 1)).slice(-2);
				var day = ('0' + date.getDate()).slice(-2);
				return year + '-' + month + '-' + day;
			},
			// 关闭KPI面板
			closeKpiPanel() {
				var vm = this;
				vm.kpiPanelShow = false;
				vm.kpiMode = false;
				vm.restoreNormalRender(); // 退出KPI模式，恢复常规渲染
			},
			// 保存当前渲染状态
			saveCurrentRenderState() {
				var vm = this;
				// 保存当前的form状态（用于恢复）
				vm._savedRenderState = {
					form: JSON.parse(JSON.stringify(vm.form)),
					statusForm: JSON.parse(JSON.stringify(vm.statusForm)),
					statusType: vm.statusType
				};
			},
			// 选择KPI日期
			selectKpiDate(date) {
				var vm = this;
				vm.kpiSelectedDate = date;
				vm.getKpiNodeData(); // TODO: 根据选择的日期加载KPI数据
				// vm.renderKpiNodes();
			},
			// Display选项变化
			onKpiDisplayChange() {
				var vm = this;
				vm.getKpiNodeData(); // TODO: 根据选择的Display类型加载相应的KPI数据
				// vm.renderKpiNodes();
			},
			// 时间范围变化
			onKpiTimeChange() {
				var vm = this;
				vm.getKpiNodeData(); // TODO: 根据时间范围过滤KPI数据
				// vm.renderKpiNodes();
			},
			// Display value复选框变化
			onKpiShowValueChange() {
				var vm = this;
				vm.getKpiNodeData();
				// vm.renderKpiNodes();
			},
			// 获取KPI数据
			getKpiNodeData() {
				var vm = this,
					ids = vm.groupIds,
					url = '${ctx}/cell/cpeinfos/getDeviceStatistics.action',
					type = vm.statusForm.deviceType.filter(function(item){
						return ['enb','gnb'].includes(item);
					}).join(',');
				
				// 处理24时的情况：24:00应该转换为第二天的00:00
				var endTime;
				if (vm.kpiSelectedTime === 24) {
					// 计算第二天的日期
					var currentDate = new Date(vm.kpiSelectedDate + ' 00:00:00');
					currentDate.setDate(currentDate.getDate() + 1);
					var nextDay = currentDate.getFullYear() + '-' + 
						('0' + (currentDate.getMonth() + 1)).slice(-2) + '-' + 
						('0' + currentDate.getDate()).slice(-2);
					endTime = nextDay + ' 00:00:00';
				} else {
					endTime = vm.kpiSelectedDate + ' ' + ('0' + vm.kpiSelectedTime).slice(-2) + ':00:00';
				}
				
				var params = {
					groupId: ids,
					timeZone: timeZone,
					deviceType: type,
					displayType: vm.kpiDisplay,
					endTime: endTime,
					startTime: vm.kpiSelectedDate + ' 00:00:00'
				};
				
				// 获取enb节点
				axios.post(url,stringify(params)).then(function(res){
					
					var kpiList = res.data || [],
						keyRef = {
							ueCount: 'ue_count',
							throughputDL: 'counter_value',
							throughputUL: 'counter_value'
						},
						kpiKey = keyRef[vm.kpiDisplay] || 'ue_count';

					kpiList = kpiList.filter(function(item){
						return item.statistics_time === params.endTime;
					});

					try{
						vm.allNodes.map(function(item){
							if(item.type == 'enb' && item.isGSM != true || item.type == 'gnb' || (item.type == 'gsm' && vm.kpiDisplay == 'ueCount')){
								// 查找对应的KPI数据
								var kpiData = kpiList.find(function(kpiItem){
									return kpiItem.device_code === item.cellCode;
								});
								if(kpiData){
									// 将KPI数据合并到节点对象中
									item.kpiValue = kpiData[kpiKey];
								}else {
									item.kpiValue = null;
								}
							}
						});
						vm.renderKpiNodes();
					}catch(e){}
				});
			},
			// 渲染KPI节点
			renderKpiNodes() {
				var vm = this,
					eNodes = vm.enbNodes.concat(vm.gnbNodes).concat(vm.gsmNodes).concat(vm.cpeNodes);
				
				if (!vm.kpiMode || !window.globOLHelper) return;

				// 处理基站数据格式
				var resObj = vm.proccessNodes(eNodes);
				vm.allNodes = resObj.enb;
				
				// 获取所有节点
				var allNodes = vm.allNodes || [];
				
				// 使用OpenLayers Helper更新节点样式
				if (window.globOLHelper.updateNodeStyles) {
					// 为每个节点生成KPI样式数据
					var nodeStyles = {};
					allNodes.forEach(function(node) {
						var nodeCode = node.code || node.cellCode || node.sn;
						var nodeType = vm.getNodeType(node);
						
						// 只对 enb 和 gnb 应用 KPI 颜色等级
						if (nodeType === 'enb' || nodeType === 'gnb' || (nodeType === 'gsm' && vm.kpiDisplay == 'ueCount')) {
							var kpiValue = node.kpiValue; //vm.getMockKpiValue(node);
							
							// 检查值是否为空、null、undefined 或 '-'
							var isEmptyValue = kpiValue === null || kpiValue === undefined || kpiValue === '';
							var isDashValue = kpiValue === '-';
							
							if (isEmptyValue || isDashValue) {
								// 空值或'-'都显示为灰色
								nodeStyles[nodeCode] = {
									color: '#999999',
									// 空值不显示，'-'显示'-'
									value: vm.kpiShowValue ? (isDashValue ? '-' : null) : null,
									nodeType: nodeType
								};
							} else {
								// 有效值根据等级着色
								var level = vm.getKpiLevel(kpiValue);
								nodeStyles[nodeCode] = {
									color: level.color,
									value: vm.kpiShowValue ? kpiValue : null,
									nodeType: nodeType
								};
							}
						} else {
							// cpe 和 gsm 固定显示为灰色
							nodeStyles[nodeCode] = {
								color: '#999999',
								value: null,
								nodeType: nodeType
							};
						}
					});
					// 批量更新所有节点样式
					window.globOLHelper.updateNodeStyles(nodeStyles, vm.kpiShowValue);
				} else {
					// 降级方案：逐个更新节点
					allNodes.forEach(function(node) {
						var nodeType = vm.getNodeType(node);
						
						if (nodeType === 'enb' || nodeType === 'gnb') {
							var kpiValue = node.kpiValue; //vm.getMockKpiValue(node);
							
							// 检查值是否为空、null、undefined 或 '-'
							var isEmptyValue = kpiValue === null || kpiValue === undefined || kpiValue === '';
							var isDashValue = kpiValue === '-';
							
							if (isEmptyValue || isDashValue) {
								// 空值或'-'都显示为灰色
								// 空值不显示，'-'显示'-'
								vm.updateNodeKpiStyle(node, { color: '#999999' }, isDashValue ? '-' : null);
							} else {
								// 有效值根据等级着色
								var level = vm.getKpiLevel(kpiValue);
								vm.updateNodeKpiStyle(node, level, kpiValue);
							}
						} else {
							// cpe 和 gsm 固定显示为灰色
							vm.updateNodeKpiStyle(node, { color: '#999999' }, null);
						}
					});
				}
			},
			// 获取节点类型
			getNodeType(node) {
				if (!node) return 'enb';
				
				// 根据节点属性判断类型
				if (node.isGnb || node.type === 'gNB' || node.type === 'gnb') return 'gnb';
				if (node.isCpe || node.type === 'CPE' || node.type === 'cpe') return 'cpe';
				// GSM 使用 isGSM 属性（注意大小写）
				if (node.isGSM === true || node.type === 'GSM' || node.type === 'gsm') return 'gsm';
				
				// 默认为 enb
				return 'enb';
			},
			// 获取模拟KPI数据（实际应从后端API获取）
			getMockKpiValue(node) {
				var vm = this;
				// 随机返回一些空值或'-'来模拟无数据情况（约20%概率）
				if (Math.random() < 0.2) {
					return Math.random() < 0.5 ? null : '-';
				}
				// 根据不同的display类型返回不同范围的模拟数据
				if (vm.kpiDisplay === 'ueCount') {
					return node.ueCount;
				} else if (vm.kpiDisplay === 'throughputDL') {
					return node.throughputDL;
				} else if (vm.kpiDisplay === 'throughputUL') {
					return node.throughputUL;
				}
				return '';
			},
			// 根据KPI值获取等级
			getKpiLevel(value) {
				var vm = this;
				var levels = vm.displayKpiLevels; // 使用计算属性获取当前显示类型对应的等级配置
				
				for (var i = 0; i < levels.length; i++) {
					var level = levels[i];
					if (value >= level.range[0] && value < level.range[1]) {
						return level;
					}
				}
				// 如果值>=最大值，返回最高等级
				return levels[levels.length - 1];
			},
			// 更新节点KPI样式
			updateNodeKpiStyle(node, level, value) {
				var vm = this;
				
				if (!window.globOLHelper) return;
				
				var nodeCode = node.code || node.cellCode || node.sn;
				
				// 尝试通过OpenLayers Helper更新单个节点
				if (window.globOLHelper.updateSingleNodeStyle) {
					window.globOLHelper.updateSingleNodeStyle(nodeCode, {
						color: level.color,
						value: vm.kpiShowValue ? value : null
					});
					return;
				}
				
				// 降级方案：直接操作DOM
				// 查找节点的canvas元素或marker元素
				var $map = $('#map');
				if ($map.length === 0) return;
				
				// OpenLayers通常使用canvas渲染，我们需要通过feature来更新
				// 这里使用CSS类名和data属性来查找相关元素
				var selector = '[data-node-code="' + nodeCode + '"]';
				var $nodeEl = $(selector);
				
				if ($nodeEl.length === 0) {
					// 尝试其他选择器
					$nodeEl = $('[data-code="' + nodeCode + '"]');
				}
				
				if ($nodeEl.length > 0) {
					// 更新节点颜色
					$nodeEl.css('color', level.color);
					$nodeEl.find('.node-icon, svg, i').css('fill', level.color).css('color', level.color);
					
					// 显示或隐藏值
					if (vm.kpiShowValue) {
						vm.showNodeKpiValue($nodeEl, value);
					} else {
						vm.hideNodeKpiValue($nodeEl);
					}
				}
			},
			// 显示节点KPI值
			showNodeKpiValue($nodeEl, value) {
				// 在节点上方显示值
				if ($nodeEl && $nodeEl.length > 0) {
					// 移除旧的值标签
					$nodeEl.find('.node-kpi-value').remove();
					
					// 添加新的值标签
					var $valueLabel = $('<div class="node-kpi-value">' + value + '</div>');
					$nodeEl.append($valueLabel);
				}
			},
			// 隐藏节点KPI值
			hideNodeKpiValue($nodeEl) {
				if ($nodeEl && $nodeEl.length > 0) {
					$nodeEl.find('.node-kpi-value').remove();
				}
			},
			// 恢复常规渲染
			restoreNormalRender() {
				var vm = this;
				
				// 清除所有KPI值显示
				$('.node-kpi-value').remove();
				
				// 如果有OpenLayers helper，通知它清除KPI模式
				if (window.globOLHelper && window.globOLHelper.clearKpiMode) {
					window.globOLHelper.clearKpiMode();
				}
				
				// 恢复部分渲染状态，但保留用户在KPI模式下对Setting的修改
				if (vm._savedRenderState) {
					// 只恢复form和statusType，不恢复statusForm（保留最新的Setting状态）
					vm.form = JSON.parse(JSON.stringify(vm._savedRenderState.form));
					vm.statusType = vm._savedRenderState.statusType;
					vm._savedRenderState = null;
				}
				// 同步全局状态到最新的statusForm
				vm.$nextTick(function() {
					// 更新全局UE状态设置（使用当前最新的statusForm）
					window.topoUeStatusSettings = vm.statusForm.ueStatus || [];
					// 更新全局Name状态设置（使用当前最新的statusForm）
					window.topoNameStatusSettings = vm.statusForm.nameStatus || [];
					// 更新全局statusForm配置，用于判断节点状态显示优先级
					window.topoStatusForm = vm.statusForm;
				});
				// 重新渲染节点，应用最新的Setting状态
				vm.getTopoInfos(vm.groupIds);
			},
			// 打开批量输入设备SN弹窗: 输入框支持以换行、空格、分号或逗号分隔多个SN
			showBatchDL() {
				var vm = this;
				
				vm.batchSnDialog = true;
				vm.batchSnForm.serialNumber = '';
			},
			closeBatchSn() {
				var vm = this;
				
				vm.batchSnDialog = false;
				// 重置表单
				if (vm.$refs.batchSnForm) {
					vm.$refs.batchSnForm.resetFields();
				}
			},
			saveBatchSn() {
				var vm = this;
				
				vm.$refs.batchSnForm.validate((valid) => {
					if (valid) {
						// 处理输入的批量SN
						var inputText = vm.batchSnForm.serialNumber || '';
						var snArray = inputText.split(/[\s,;]+/).map(sn => sn.trim()).filter(sn => sn !== '');
						// 发送批量导入请求
						axios.post('${ctx}/fence/batchBindDevices', {
							fenceId: vm.currentFenceConfig.originalFenceId,
							deviceSns: snArray.join(';')
						}).then(function(res){
							var data = res.data || {};
							// 处理返回结果
							var successList = (data.successDevices || []).map(d => ({ code: d.serialNumber, cellName: d.hostName, status: d.status }));
							var failList = data.failedDevices || [];
							// 成功结果更新到修改页面的设备列表中，根据SN去重
							successList.forEach(function(device){
								if (!vm.currentFenceConfig.deviceList.find(d => d.code === device.code)) {
									vm.currentFenceConfig.deviceList.push(device);
								}
							});
							vm.closeBatchSn();
							// failList不为空时，弹出dialog显示失败的SN
							if (failList.length > 0) {
								vm.batchSnErrorForm.errorSerialNumber = failList.join('\n');
								vm.batchSnErrorDialog = true;
							}
						}).catch(function(error){ });
					}
				});
			},
			downloadErrorSns() {
				var vm = this;
				
				// 下载失败的SN列表为txt文件
				var failSns = vm.batchSnErrorForm.errorSerialNumber;
				var blob = new Blob([failSns], { type: 'text/plain;charset=utf-8' });
				var link = document.createElement('a');
				link.href = window.URL.createObjectURL(blob);
				link.download = 'failed_sns.txt';
				document.body.appendChild(link);
				link.click();
				document.body.removeChild(link);
			},
			closeBatchSnError() {
				var vm = this;
				
				vm.batchSnErrorDialog = false;
				vm.batchSnErrorForm.errorSerialNumber = '';
			},
			// 设置fence开关
			changeFenceEnable() {
				var vm = this;
				
				if (vm.fenceEnable) {
					axios.post('${ctx}/fence/operatorSwitch', { switch: 0 })
					.then(function(res){
						var data = res.data;
						if(data && data.success == true) {
							vm.$message.success(data.message);

							// 关闭围栏显示时，清除地图上的围栏显示
							if (window.globOLHelper) {
								window.globOLHelper.clearAllFences();
							}
							vm.fenceEnable = !vm.fenceEnable;
						} else {
							vm.$message.error(data.message);
						}
					}).catch(function(error){ });
				} else {
					var msg = '<%=rb.getString("DianZiWeiLanQiYongTiShi")%>';
					vm.$confirm(msg, '<%=rb.getString("QueRen")%>', {
						confirmButtonText: '<%=rb.getString("QueDing")%>',
						cancelButtonText: '<%=rb.getString("QuXiao")%>',
						type: 'warning'
					}).then(() => {
						axios.post('${ctx}/fence/operatorSwitch', { switch: 1 })
						.then(function(res){
							var data = res.data;
							if(data && data.success == true) {
								vm.$message.success(data.message);
								
								vm.fenceEnable = !vm.fenceEnable;
								// 回显围栏
								vm.reviewFence();
							} else {
								vm.$message.error(data.message);
							}
						}).catch(function(error){ });
					}).catch(() => {});
				}
			},
			fenceStatusChange(row) {
				var vm = this;
				
				axios.post('${ctx}/fence/updateFenceStatus', {
					fenceId: row.id,
					fenceStatus: row.enable ? 1 : 0
				}).then(function(res){
					var data = res.data;
					if(data && data.success == true) {
						vm.$message.success(data.message);
						// 回显围栏
						vm.reviewFence();
					} else {
						vm.$message.error(data.message);
						row.enable = !row.enable; // 恢复原状态
					}
				}).catch(function(error){
					row.enable = !row.enable; // 恢复原状态
				});
			},
			// 绘制电子围栏
			drawFence() {
				var vm = this;
				// 检查是否启用围栏功能
				if(!vm.fenceEnable) {
					return;
				}
				vm.opType = 'add';
				// 检查地图是否已初始化
				if (!window.globOLHelper || !window.globOLHelper.map) {
					return;
				}
				vm.closeFenceView();
				vm.currentFenceConfig = {
					enable: false,
					fenceName: '',
					deviceList: [],
					fenceParams: null,
					feature: null,
					originalFenceId: null
				};
				
				vm.startDrawing();
				document.body.click();
			},
			
			// 获取fence全局开关状态
			getFenceEnableStatus() {
				var vm = this;
				
				axios.post('${ctx}/fence/queryOperatorSwitch')
				.then(function(response) {
					var data = response.data || {};
					
					vm.fenceEnable = data.fenceSwitch == true ? true : false;
					
					vm.loadFenceDataFromStorage();
				});
			},
			// 从 localStorage 加载围栏数据（仅加载数据，不在地图上显示）
			loadFenceDataFromStorage() {
				var vm = this;
				
				axios.post('${ctx}/fence/queryFenceList')
				.then(function(response) {
					if (response.data && Array.isArray(response.data.rows)) {
						var list = response.data.rows || [];

						vm.fenceList = list.map(function(fence) {
							return {
								id: fence.fenceId,
								name: fence.fenceName,
								type: 'Polygon',
								coordinates: fence.fenceCoords,
								enable: fence.fenceStatus == 1 ? true : false,
								deviceList: fence.deviceList || []
							};
						});

						// 如果启用围栏，回显围栏
						if (vm.fenceEnable) {
							vm.reviewFence();
						}
					} else {
						vm.fenceList = [];
					}
				});
			},
			
			// 开始绘制
			startDrawing() {
				var vm = this,
					drawType = 'Polygon';
				
				// 创建绘制交互
				var drawInteraction = window.globOLHelper.createDrawInteraction(drawType, function(fenceParams, feature) {
					
					// 确保 fenceList 存在
					if (!vm.fenceList) {
						vm.fenceList = [];
					}
					
					// 使用保存的原ID（编辑模式下redraw时）或生成新ID
					if (vm.currentFenceConfig.originalFenceId) {
						fenceParams.id = vm.currentFenceConfig.originalFenceId;
					} else {
						fenceParams.id = 'fence_' + Date.now();
					}
					
					// 检测围栏内的设备
					var geometry = feature.getGeometry();
					var devicesInFence = window.globOLHelper.getDevicesInFence(geometry, vm.allNodes || []);
					
					// 准备配置面板数据
					Object.assign(vm.currentFenceConfig, {
						deviceList: devicesInFence.filter(function(device) {
							return device.type === 'enb' && device.fullData.isGSM !== true;
						}).map(function(device) {
							return {
								serialNumber: device.code,
								cellName: device.cellName || device.fullData.name,
								type: 'enb',
								code: device.code,
								status: device.fullData.online
							};
						}),
						fenceParams: fenceParams,
						feature: feature // 保存feature引用，用于后续操作
					});
					
					// 显示配置面板
					vm.fenceConfigVisible = true;
				});
			},
			// 重新绘制电子围栏
			redrawFence() {
				var vm = this;
				
				// 检查地图是否已初始化
				if (!window.globOLHelper || !window.globOLHelper.map) {
					return;
				}

				// 保存原围栏的ID（用于编辑模式）
				var originalFenceId = vm.currentFenceConfig.fenceParams ? vm.currentFenceConfig.fenceParams.id : null;

				// 从地图上移除围栏图形
				if (vm.currentFenceConfig.feature && window.globOLHelper && window.globOLHelper.fenceSource) {
					window.globOLHelper.fenceSource.removeFeature(vm.currentFenceConfig.feature);
				}

				// 清空当前配置但保留围栏名称和enable状态
				var savedFenceName = vm.currentFenceConfig.fenceName;
				var savedEnable = vm.currentFenceConfig.enable;
				
				Object.assign(vm.currentFenceConfig, {
					enable: savedEnable,
					fenceName: savedFenceName,
					deviceList: [],
					fenceParams: null,
					feature: null,
					originalFenceId: originalFenceId // 保存原ID用于后续更新
				});
				// 关闭配置面板
				vm.fenceConfigVisible = false;

				vm.startDrawing();
			},
			// 修复电子围栏
			modifyFence(row) {
				var vm = this;
				
				// 检查是否启用围栏功能
				if(!vm.fenceEnable) {
					return;
				}

				vm.opType = 'edit';

				// 检查地图是否已初始化
				if (!window.globOLHelper || !window.globOLHelper.map) {
					return;
				}

				// 从地图上移除围栏图形
				try {
					window.globOLHelper.removeFenceById(row.id);
				} catch (e) {
					console.error('移除围栏失败:', e);
				}
				vm.closeFenceView();
				vm.deviceQueryStr = '';

				var url = '${ctx}/fence/queryDevicesByFence?rd=' + Math.random();
				axios.post(url, { fenceId: row.id })
				.then(function(response) {
					var data = response.data || {};
					
					// 更新围栏的设备列表
					row.deviceList = (data.rows || []).map(function(device) {
						return {
							serialNumber: device.serialNumber,
							cellName: device.hostName,
							type: device.neType,
							code: device.serialNumber,
							status: device.status
						};
					});
					
					// 准备配置面板数据
					vm.currentFenceConfig = {
						enable: row.enable,
						fenceName: row.name,
						deviceList: row.deviceList,
						fenceParams: {
							id: row.id,
							name: row.name,
							type: row.type,
							coordinates: row.coordinates
						},
						originalFenceId: null  // 修改模式下不使用redraw则不需要originalFenceId
					};
					
					// 显示配置面板
					vm.fenceConfigVisible = true;

					// 重新绘制围栏以供修改
					try {
						var feature = window.globOLHelper.drawFenceFromParams(vm.currentFenceConfig.fenceParams);
						vm.currentFenceConfig.feature = feature; // 保存feature引用，用于后续操作
					} catch (e) {
						console.error('绘制围栏失败:', e);
					}
					document.body.click();
				}).catch(function(error) { });
			},
			removeFence(row) {
				var vm = this;

				// 检查是否启用围栏功能
				if(!vm.fenceEnable) {
					return;
				}
				
				var msg = '<%=rb.getString("DianZiWeiLanShanChuTiShi")%>';
				vm.$confirm(msg, '<%=rb.getString("QueRen")%>', {
					confirmButtonText: '<%=rb.getString("ShanChu")%>',
					cancelButtonText: '<%=rb.getString("QuXiao")%>',
					type: 'warning'
				}).then(() => {
					var params = {
							operateType: 3, // 删除操作
							fenceId: row.id
						};
					// 发送删除请求到后端
					axios.post('${ctx}/fence/manage', params)
					.then(function(response) {
						var data = response.data || {};
						if (data.success) {
							vm.$message.success(data.message);
							// 从地图上移除围栏图形
							try {
								window.globOLHelper.removeFenceById(row.id);
							} catch (e) {
								console.error('移除围栏失败:', e);
							}
							
							// 从 fenceList 中移除
							vm.fenceList = vm.fenceList.filter(function(fence) {
								return fence.id !== row.id;
							});
						} else {
							vm.$message.warning(data.message);
						}
					}).catch(function(error) { });
				}).catch(() => {
					// 取消操作
				});
			},
			/**
			 * 双击围栏列表行事件处理
			 * @param row{object}: 围栏数据行
			 */
			fenceRowDblClick(row) {
				var vm = this;

				// 检查围栏是否启用
				if (row.enable !== true) {
					return;
				}

				// 检查地图和围栏助手是否已初始化
				if (!window.globOLHelper || !window.globOLHelper.map) {
					return;
				}

				try {
					// 调用地图助手的定位方法
					window.globOLHelper.locateToFence(row.id);
				} catch (e) {
					console.error('定位到围栏失败:', e);
				}
			},
			removeDeviceFromFence(row) {
				var vm = this;
				
				// 从当前围栏配置的设备列表中移除
				vm.currentFenceConfig.deviceList = vm.currentFenceConfig.deviceList.filter(function(device) {
					return device.code !== row.code;
				});
			},
			
			// 回显电子围栏
			reviewFence() {
				var vm = this;
				
				// 检查地图是否已初始化
				if (!window.globOLHelper || !window.globOLHelper.map) {
					return;
				}

				// 清除现有围栏显示
				window.globOLHelper.clearAllFences();
				
				// 检查是否有围栏数据
				if (!vm.fenceList || vm.fenceList.length === 0) {
					return;
				}
				
				// 在地图上绘制所有围栏
				var successCount = 0;
				vm.fenceList.forEach(function(fenceParams) {
					try {
						if(fenceParams.enable === true) {
							window.globOLHelper.drawFenceFromParams(fenceParams);
							successCount++;
						}
					} catch (e) {
						console.error('绘制围栏失败:', e, fenceParams);
					}
				});
			},
			
			// 确认围栏配置
			confirmFenceConfig() {
				var vm = this;
				
				// 验证围栏名称
				if (!vm.currentFenceConfig.fenceName || vm.currentFenceConfig.fenceName.trim() === '') {
					vm.$message.warning('<%=rb.getString("WeiLanMingChengJiaoYan")%>');
					return;
				}
				
				// 准备保存的围栏数据
				var fenceData = {
					id: vm.currentFenceConfig.fenceParams.id,
					name: vm.currentFenceConfig.fenceName,
					enable: vm.currentFenceConfig.enable,
					type: vm.currentFenceConfig.fenceParams.type,
					coordinates: vm.currentFenceConfig.fenceParams.coordinates,
					deviceList: vm.currentFenceConfig.deviceList.map(function(device) {
						return {
							serialNumber: device.serialNumber,
							cellName: device.cellName,
							type: device.type,
							code: device.code,
							status: device.status
						};
					}),
					createTime: new Date().toISOString()
				};
				
				// 确保 fenceList 存在
				if (!vm.fenceList) {
					vm.fenceList = [];
				}

				// 准备发送到后端的参数
				var params = {
						operateType: vm.opType === 'edit'? 2:1, // 2表示更新
						fenceId: vm.opType === 'edit'? fenceData.id : null,
						fenceName: fenceData.name,
						fenceStatus: fenceData.enable? 1:0,
						type: fenceData.type,
						fenceCoords: fenceData.coordinates,
						devices: fenceData.deviceList.map(function(device) {
									return device.code;
								})
					};
				// 发送保存请求到后端
				axios.post('${ctx}/fence/manage', params)
				.then(function(response) {
					var data = response.data || {};
					if (data.success) {
						vm.$message.success('<%=rb.getString("ChengGong")%>');
						
						if(vm.opType == 'edit') {
							// 编辑模式：查找并更新原有数据
							var index = vm.fenceList.findIndex(function(fence) {
								return fence.id === fenceData.id;
							});
							
							if (index !== -1) {
								// 保留原创建时间，更新其他信息
								fenceData.createTime = vm.fenceList[index].createTime;
								vm.fenceList.splice(index, 1, fenceData);
							} else {
								// 如果没找到（可能是redraw后的新ID），添加为新数据
								vm.fenceList.push(fenceData);
							}
						} else {
							// 新增模式：直接添加到围栏列表
							vm.fenceList.push(fenceData);
						}
						// 关闭配置面板
						vm.fenceConfigVisible = false;
						
						// 清空当前配置
						vm.currentFenceConfig = {
							enable: false,
							fenceName: '',
							deviceList: [],
							fenceParams: null,
							originalFenceId: null
						};
						
						// 清空操作类型
						vm.opType = '';
						vm.loadFenceDataFromStorage();
					} else {
						vm.$message.error(data.message);
					}
				});
			},
			
			// 取消围栏配置
			cancelFenceConfig() {
				var vm = this;
				
				if(vm.opType == 'edit') {
					// 编辑模式下取消，需要恢复原始围栏的显示
					
					// 移除当前绘制的图形（使用 feature 直接移除）
					if (vm.currentFenceConfig.feature && window.globOLHelper && window.globOLHelper.fenceSource) {
						try {
							window.globOLHelper.fenceSource.removeFeature(vm.currentFenceConfig.feature);
						} catch (e) {}
					}
					
					// 恢复原始围栏的显示
					var fenceId = vm.currentFenceConfig.originalFenceId || vm.currentFenceConfig.fenceParams.id;
					var originalFence = vm.fenceList.find(function(fence) {
						return fence.id === fenceId;
					});
					
					if (originalFence && vm.fenceEnable) {
						// 重新绘制原始围栏
						try {
							window.globOLHelper.drawFenceFromParams(originalFence);
						} catch (e) {}
					}
					
					vm.fenceConfigVisible = false;
					vm.currentFenceConfig = {
						enable: false,
						fenceName: '',
						deviceList: [],
						fenceParams: null,
						feature: null,
						originalFenceId: null
					};
					vm.opType = '';
					vm.loadFenceDataFromStorage();
					return;
				}

				vm.$confirm('<%=rb.getString("QuXiaoWeiLanTiShi")%>', '<%=rb.getString("QueRen")%>', {
					confirmButtonText: '<%=rb.getString("QueDing")%>',
					cancelButtonText: '<%=rb.getString("FangHui")%>',
					type: 'warning'
				}).then(() => {
					// 从地图上移除围栏图形
					if (vm.currentFenceConfig.feature && window.globOLHelper && window.globOLHelper.fenceSource) {
						window.globOLHelper.fenceSource.removeFeature(vm.currentFenceConfig.feature);
					}
					
					// 关闭配置面板
					vm.fenceConfigVisible = false;
					
					// 清空当前配置
					vm.currentFenceConfig = {
						enable: false,
						fenceName: '',
						deviceList: [],
						fenceParams: null,
						feature: null,
						originalFenceId: null
					};
					
					// 清空操作类型
					vm.opType = '';
					vm.loadFenceDataFromStorage();
				}).catch(() => {
					// 用户选择继续编辑，不做任何操作
				});
			},
			
			// 获取设备类型对应的标签颜色
			getDeviceTypeColor(type) {
				var colorMap = {
					'eNB': 'success',    // 绿色
					'gNB': 'primary',    // 蓝色
					'CPE': 'info',       // 灰色
					'WCG': 'warning'     // 橙色
				};
				return colorMap[type] || '';
			},

			// 导出围栏设备列表
			exportFenceDevices() {
				var vm = this;
				
				// 构建导出URL
				exportByForm('${ctx}/fence/exportFenceDevices', {
					fenceId: vm.fenceDeviceQuery.fenceId
				});
			},
			// 打开查看围栏设备浮层
			showFenceDevices(fenceParams) {
				var vm = this;
				
				// 显示查看面板
				vm.fenceViewVisible = true;

				Object.assign(vm.fenceDeviceQuery, {
					fenceId: fenceParams.id,
					searchText: '',
					fenceName: fenceParams.name
				});
				var url = '${ctx}/fence/queryDevicesByFence?rd=' + Math.random();
				axios.post(url, {
					fenceId: fenceParams.id
				}).then(function(res){
					var data = res.data || {};
					vm.fenceDeviceList = data.rows || [];
					
				}).catch(function(error){ });
			},
			// 关闭查看围栏设备浮层
			closeFenceView() {
				var vm = this;
				vm.fenceViewVisible = false;
				Object.assign(vm.fenceDeviceQuery, {
					fenceId: '',
					searchText: '',
					fenceName: ''
				});
			},

			// 切换搜索面板
			toggleSearchPanel() {
				var vm = this;
				vm.searchPanelShow = !vm.searchPanelShow;
				if (vm.searchPanelShow) {
					vm.clusterPanelShow = false; // 关闭重叠节点面板
				}
				vm.searchKeyword = '';
				vm.searchResults = [];
			},
			// 处理搜索
			handleSearch() {
				var vm = this;
				if (!vm.searchKeyword || vm.searchKeyword.trim() === '') {
					vm.searchResults = [];
					return;
				}
				
				// 使用 globOLHelper 的搜索方法
				if (window.globOLHelper && vm.allNodes && vm.allNodes.length > 0) {
					vm.searchResults = window.globOLHelper.searchNodes(vm.searchKeyword, vm.allNodes);
				} else {
					vm.searchResults = [];
				}
			},
			// 智能定位节点（自动判断是否需要保持展开状态）
			locateNodeSmart(node) {
				var vm = this;
				if (!window.globOLHelper) return;
				
				// 默认情况下，查询结果点击时应该展开并保持展开状态
				var shouldKeepExpanded = false;
				
				// 情况1：检查该节点是否属于当前展开的 cluster
				if (window.globOLHelper.expandedCluster && window.globOLHelper.expandedFeatures) {
					// 检查展开的节点中是否包含该节点
					var isInExpandedCluster = window.globOLHelper.expandedFeatures.some(function(feature) {
						var nodeData = feature.get('nodeData');
						return nodeData && nodeData.code === node.code;
					});
					
					if (isInExpandedCluster) {
						shouldKeepExpanded = true;
					}
				}
				
				// 情况2：检查该节点是否在某个cluster中（未展开的情况）
				if (!shouldKeepExpanded && node.lat && node.lon) {
					var locKey = node.lat + ',' + node.lon;
					var clusterNodes = window.globOLHelper.clusterMap.get(locKey);
					
					// 如果该位置有多个节点，说明是cluster，定位后应该保持展开
					if (clusterNodes && clusterNodes.length > 1) {
						shouldKeepExpanded = true;
					}
				}
				
				// 调用定位方法
				vm.locateNode(node, shouldKeepExpanded);
			},
			// 定位到指定节点
			locateNode(node, keepExpanded) {
				var vm = this;
				if (window.globOLHelper) {
					// 注意：locateAndHighlightNode 会自动检测并展开cluster
					// 所以这里不需要手动收起cluster，让方法自己处理
					window.globOLHelper.locateAndHighlightNode(node, 18, keepExpanded);
					
					// 关闭搜索面板（无论是否保持展开）
					vm.searchPanelShow = false;
					
					// 如果不保持展开状态，则关闭cluster面板
					if (!keepExpanded) {
						vm.clusterPanelShow = false;
						// 注意：不再强制收起cluster，因为locateAndHighlightNode可能刚展开它
						// 只有在点击空白区域或用户明确操作时才收起
					}
				}
			},
			// 显示重叠节点列表
			showClusterNodes(nodes) {
				var vm = this;
				vm.clusterNodes = nodes;
				vm.clusterPanelShow = true;
				vm.searchPanelShow = false; // 关闭搜索面板
			},
			// 获取节点类型文本
			getNodeTypeText(type) {
				var typeMap = {
					'enb': 'eNB',
					'gnb': 'gNB',
					'cpe': 'CPE',
					'gsm': 'GSM'
				};
				return typeMap[type] || type;
			},
			sortOverview() {
				var vm = this;

				if(vm.overViewList && vm.overViewList.length) {
					vm.overViewList = vm.overViewList.sort(function(a, b){
						return b.total - a.total;
					})
				}
			},
			initOverView() {
				var vm = this,
					url = '${ctx}/cell/cpeinfos/getDeviceGroupNames.action?rd=' + Math.random();

				if(vm.topoOverViewEnable !== true) {
					return;
				}

				vm.overViewList = [];
				axios.post(url).then((res)=>{
					var list = res.data || [];

					list.map(function(code){
						vm.overViewList.push({
							area: code,
							btsTotal: '',
							btsOnline: '',
							enbTotal: '',
							enbOnline: '',
							gnbTotal: '',
							gnbOnline: '',
							onlineTotal: '',
							total: ''
						})
					})

					// 获取 BTS Total
					axios.post('${ctx}/cell/cpeinfos/getBtsTotal.action?rd=' + Math.random()).then((res)=>{
						var data = res.data;

						vm.overViewList.map(function(item){
							item.btsTotal = data[item.area];
						})
						vm.staticTotal();
					})
					// 获取 BTS Online
					axios.post('${ctx}/cell/cpeinfos/getBtsOnlineCount.action?rd=' + Math.random()).then((res)=>{
						var data = res.data;

						vm.overViewList.map(function(item){
							item.btsOnline = data[item.area];
						})
						vm.staticTotal();
					})
					// 获取 eNB Total
					axios.post('${ctx}/cell/cpeinfos/getEnbTotal.action?rd=' + Math.random()).then((res)=>{
						var data = res.data;

						vm.overViewList.map(function(item){
							item.enbTotal = data[item.area];
						})
						vm.staticTotal();
					})
					// 获取 eNB Online
					axios.post('${ctx}/cell/cpeinfos/getEnbOnlineCount.action?rd=' + Math.random()).then((res)=>{
						var data = res.data;

						vm.overViewList.map(function(item){
							item.enbOnline = data[item.area];
						})
						vm.staticTotal();
					})
					// 获取 gNB Online
					axios.post('${ctx}/cell/cpeinfos/getGnbTotal.action?rd=' + Math.random()).then((res)=>{
						var data = res.data;

						vm.overViewList.map(function(item){
							item.gnbTotal = data[item.area];
						})
						vm.staticTotal();
					})
					// 获取 gNB Total
					axios.post('${ctx}/cell/cpeinfos/getGnbOnlineCount.action?rd=' + Math.random()).then((res)=>{
						var data = res.data;

						vm.overViewList.map(function(item){
							item.gnbOnline = data[item.area];
						});
						vm.staticTotal();
					})
				})
			},
			staticTotal() {
				var vm = this;

				vm.overViewList.map(function(item){
					item.onlineTotal = ((item.btsOnline||0) - 0) + ((item.enbOnline||0) - 0) + ((item.gnbOnline||0) - 0);
					item.total = ((item.btsTotal||0) - 0) + ((item.enbTotal||0) - 0) + ((item.gnbTotal||0) - 0);
				});
				vm.sortOverview();
			},
			confirmMove() {
				var vm = this,
					form = vm.gpsmvoeForm,
					olatlng = form.olatlng;

				$.ajax({
					url: '${ctx}/cell/topo/setLocationInfo.action',
					type: 'post',
					dataType: 'json',
					data: {
						longitude: form.lon,
						latitude: form.lat,
						setType: 'topo',
						cell_code: form.cell_code,
						operator_code: operator_code
					},
					success: function(data){
						if(data['success']){
							// 刷新节点数据
							vm.getTopoInfos(vm.groupIds);
							
							// 刷新设备列表
							vm.$refs.enbDevice.refresh();
							if(writableMap.CODE_CPE_MONITOR != undefined) {
								vm.$refs.cpeDevice.refresh();
							}
						}else{
							toast(data['message'],$('#omc_app_ctn'));
						}
						vm.gpsmoveVisible = false;
					}
				})
			},
			cancelMove() {
				var vm = this,
					form = vm.gpsmvoeForm,
					olatlng = form.olatlng,
					target = form.target;
				
				// 判断target类型：OpenLayers Feature 或 Leaflet Marker
				if(target) {
					// OpenLayers Feature（通过检查get方法）
					if(typeof target.get === 'function' && form.originalPosition) {
						// 使用olHelper恢复位置
						if(window.globOLHelper) {
							window.globOLHelper.restoreFeaturePosition(target, form.originalPosition);
						}
					} 
				}
				vm.gpsmoveVisible = false;
			},
			measure() {
				// 如果使用OpenLayers地图
				if (window.globOLHelper && window.globOLHelper.map) {
					// 先清除按钮状态（防止页面重新加载后状态不一致）
					$('.flex-bt-cls').removeClass('selected');					// 切换测距状态
					if (!window.globOLHelper.measureActive) {
						window.globOLHelper.startMeasure();
						$('.flex-bt-cls').addClass('selected');
					} else {
						window.globOLHelper.stopMeasure();
					}
				}
			},
			measureSite() {
				// 如果使用OpenLayers地图
				if (window.globOLHelper && window.globSiteMap) {
					// 先清除按钮状态（防止页面重新加载后状态不一致）
					$('.flex-bt-cls').removeClass('selected');
					
					// 切换测站状态
					if (!window.globOLHelper.measureSiteActive) {
						window.globOLHelper.startMeasureSite();
						$('.flex-bt-cls').addClass('selected');
					} else {
						window.globOLHelper.stopMeasureSite();
					}
				}
			},
			activeFmt(value, row) {
				value = value == 'yes'? 
								  '<i class="el-icon el-icon-status-active online margin-right-5"></i><%=rb.getString("JiHuo")%>'
								  :'<i class="el-icon el-icon-status-active inactive margin-right-5"></i><%=rb.getString("QuJiHuo")%>';

				return value;
			},
			oldMmeStatusFmt(nodeInfo,value,type){
				var vm = this,
					rowData = {
						s1siglinkserverlist: nodeInfo.serverList,
						mmeEnable: nodeInfo.mmeEnable,
						mmeStatus: nodeInfo.mmeStatus,
						mme_pool_1: nodeInfo.mmePool1,
						mme_pool_2: nodeInfo.mmePool2
					},
					mmeStatus,textVal,value,hasDisconn=false,hasConn=false,mmeDataList=[],leftStr='',parseMMEOnNum=0;
			
				if (value == null || value == "") {
					mmeStatus ='';
					textVal = '--'
				}else{
					if (value == "1") {
						mmeDataList.push({mmeIp:rowData.s1siglinkserverlist,status:'1'})
						mmeStatus ='';
						textVal = '<%= rb.getString("MMEYiLianJie")%>'
					} else if (value == "0") {
						mmeDataList.push({mmeIp:rowData.s1siglinkserverlist,status:'0'})
						mmeStatus ='offStatusCls';
						textVal = '<%= rb.getString("MMEWeiLianJie")%>'
					} else if (value == "2") {
						mmeStatus ='';
						textVal = '--'
					} else{
						var mmePools = value.split(",");
						for(var i = 0; i < mmePools.length; i++ ){
							var mme = mmePools[i];
							var mmeArr = mme.split("=");
							
							if(mmeArr.length == 2 && "mme1" == mmeArr[0]){
								mmeDataList.push({mmeIp:rowData.mme_pool_1,status:mmeArr[1]})
							}else if(mmeArr.length == 2 && "mme2" == mmeArr[0]){
								mmeDataList.push({mmeIp:rowData.mme_pool_2,status:mmeArr[1]})
							}
						}
						mmeDataList.map((item)=>{
							if (item.status == '1'){
								hasConn = true;
							}else if(item.status == '0'){
								hasDisconn = true
							}
						})
						if (hasDisconn == true){
							if (hasConn == false) {
								mmeStatus ='offStatusCls';
								textVal = '<%= rb.getString("MMEWeiLianJie")%>'
							}else{
								mmeStatus ='offOrOnStatusCls';
								textVal = '<%= rb.getString("MMEYiLianJie")%>'
							}
						}else {
							mmeStatus ='';
							textVal = '<%= rb.getString("MMEYiLianJie")%>'
						}
					}
				}
				mmeDataList.map((item)=>{
					if (item.status == '1'){
						parseMMEOnNum += 1;
					}
				})
				if(type == 'left'){
					value = "<span class='"+ mmeStatus +"'>" + textVal +"</span>"
				}else if(type == 'right'){
					value = parseMMEOnNum + '/' + mmeDataList.length
				}else if(type == 'list'){
					value = mmeDataList
				}
				return value;
			},
			mmeStatusFmt(value, nodeInfo, rowIndex) {
				var row = {
					s1siglinkserverlist: nodeInfo.serverList,
					mmeEnable: nodeInfo.mmeEnable,
					mmeStatus: nodeInfo.mmeStatus,
					mme_pool_1: nodeInfo.mmePool1,
					mme_pool_2: nodeInfo.mmePool2
				};

				return mmeStatusFormatterTopo(value, row, rowIndex);
			},
			mmesStatusFmt(str){
				var mmelist = eval('('+str+')');
				var mmeStatus,textVal,value,hasDisconn=false,hasConn=false;
				mmelist.map(function(item){
					if (item.status == '1'){
						hasConn = true;
					}else if(item.status == '0'){
						hasDisconn = true
					}
				})
				
				if (hasDisconn == true){
					if (hasConn == false) {
						mmeStatus ='offStatusCls';
						textVal = 'Disconnected'
					}else{
						mmeStatus ='offOrOnStatusCls';
						textVal = 'Connected'
					}
				}else {
					mmeStatus ='';
					textVal = 'Connected'
				}
				
				value = "<span class='"+ mmeStatus +"'>" + textVal +"</span>"
				return value;
			},
			parseMMEOnNum(str){
				var onNum = [],
					list = [];
				if(str) {
					list = eval('('+str+')');
				}else {
					list = [];
				}
				list.map((item,index)=>{
					if(item.status == '1'){
						onNum.push(item)
					}
				})
				return onNum.length
			},
			parseMME(str) {
				if(str) {
					return eval('('+str+')');
				}else {
					return [];
				}
			},

			mmeIPfmt(value, nodeInfo) {
				var ips = [];

				if(nodeInfo.mmePool1) {
					ips.push(nodeInfo.mmePool1);
				}
				if(nodeInfo.mmePool2) {
					ips.push(nodeInfo.mmePool2);
				}
				if(ips.length==0) {
					ips.push(nodeInfo.serverList);
				}

				return ips.join(', ');
			},
			alarmFmt(value, row) {
				return alarmInfoFormatter(value, row);
			},
			sasEnableFmt(value, row) {
				var vm = this,
					enable = row.sasEnable
					str = '';

				if(enable == '--') {
					str = '--';
				}else if(enable == 'on') {
					str = '<i class="el-icon el-icon-status-enable"></i>' + enable;
				}else {
					str = '<i class="el-icon el-icon-status-disable"></i>' + enable;
				}
				
				return str;
			},
			sasStateFmt(value, row) {
				var vm = this,
					state = row.sasState,
					str = '',
					codes = {
						Unregistered: 'offline',
						Registered: 'registed',
						Granted: 'grated',
						Authorized: 'authed'
					};

				if(value) {
					str = '<i class="el-icon el-icon-topo-enb margin-right-5 ' + codes[value] + '"></i>' + value;
				}

				return str;
			},
			rsrpFmt(nodeInfo) {
				return showRedAccordingRSRP(nodeInfo.RSRP0, nodeInfo) + ' ' + showRedAccordingRSRP(nodeInfo.RSRP1, nodeInfo);
			},
			statusTypeChange(code) {
				var vm = this,
					codes = {
						deviceStatus: ['on','off'],
						activeStatus: ['yes','no'],
						sasStatus: ['Unregistered','Registered','Granted','Authorized']
					};
				
				['deviceStatus','activeStatus','sasStatus'].map(function(item) {
					if(item == code) {
						vm.statusForm[item] = codes[item];
					}else {
						vm.statusForm[item] = []
					}
				});
				
				vm.$nextTick(function(){
					vm.setStatus();
				});
			},
			statusChange(code) {
				var vm = this;
				
				vm.statusType = code;
				
				['deviceStatus','activeStatus','sasStatus'].map(function(item) {
					if(item != code) {
						vm.statusForm[item] = []
					}
				});

				// 更新全局statusForm配置，用于判断节点状态显示优先级
				window.topoStatusForm = vm.statusForm;

				vm.$nextTick(function(){
					vm.setStatus();
				});
			},
			ueStatusChange(val) {
				var vm = this;

				vm.statusForm.ueStatus = val;
				
				// 更新全局变量供OpenLayers使用
				window.topoUeStatusSettings = val || [];
				// 更新全局statusForm配置，用于判断节点状态显示优先级
				window.topoStatusForm = vm.statusForm;
				
				vm.$nextTick(function(){
					vm.setStatus();
					
					// 清除样式缓存并触发地图节点样式更新，使用防抖避免频繁触发
					if (globOLHelper && globOLHelper.vectorSource) {
						if (typeof globOLHelper.clearStyleCache === 'function') {
							globOLHelper.clearStyleCache();
						}
						if (typeof globOLHelper.debouncedRefresh === 'function') {
							globOLHelper.debouncedRefresh(200);
						} else {
							globOLHelper.vectorSource.changed();
						}
					}
				});
			},
			nameStatusChange(val) {
				var vm = this;
				vm.statusForm.nameStatus = val;
				
				// 更新全局变量供OpenLayers使用
				window.topoNameStatusSettings = val || [];
				
				vm.$nextTick(function(){
					// 清除样式缓存并触发地图节点样式更新，使用防抖避免频繁触发
					if (globOLHelper && globOLHelper.vectorSource) {
						if (typeof globOLHelper.clearStyleCache === 'function') {
							globOLHelper.clearStyleCache();
						}
						if (typeof globOLHelper.debouncedRefresh === 'function') {
							globOLHelper.debouncedRefresh(200);
						} else {
							globOLHelper.vectorSource.changed();
						}
					}
				});
			},
			/**
			* 双击列表行 自动定位topo中节点（如果在tops中）
			* @param row{object}: 列表行数据
			* @param evt{event}：鼠标事件
			**/
			rowdblclick(row,evt) {
				var vm = this;
				var code = row.cpe_code||row.serial_number;
				
				// 在所有节点中查找该节点
				var targetNode = vm.allNodes.find(function(node){
					return node.code === code;
				});
				
				// topo中有该节点时，使用智能定位
				if(targetNode && targetNode.lat && targetNode.lon){
					vm.locateNodeSmart(targetNode);
					vm.selectedCode = code;
				}
			},
			/**
			* 设备组列表数据加载成功回调
			* @param req{object}: 列表加载返回数据
			**/
			loadSuccess(req) {
				var vm = this,
					rows = req.rows,
					row = '';
				
				if(rows && rows.length) row = rows[0];
				// 首次加载默认选中第一行
				if(vm.firtLoad) {
					setTimeout(function(){
						vm.firtLoad = false;
						// 设备组所有列全选
						vm.$refs.group.clearSelection();
						vm.$refs.group.toggleAllSelection();
					},10);
				}
			},
			/**
			* 设备组列表查询
			* @param text{string}: 检索文本
			**/
			queryGroup(text) {
				var vm = this;

				vm.groupQueryParams.search_text = (text||'').trim();
			},
			/**
			* 设备列表查询
			* @param text{string}: 检索文本
			**/
			queryDevice(text) {
				var vm = this,
					stxt = (text||'').trim();

				vm.enbQueryParams.search_text = stxt;
			},
			queryGNBDevice(text) {
				var vm = this,
					stxt = (text||'').trim();

				vm.gnbQueryParams.search_text = stxt;
			},
			queryGSMDevice(text) {
				var vm = this,
					stxt = (text||'').trim();

				vm.gsmQueryParams.search_text = stxt;
			},
			/**
			* 设备列表查询
			* @param text{string}: 检索文本
			**/
			queryCpeDevice(text) {
				var vm = this,
					stxt = (text||'').trim();

				vm.cpeQueryParams.search_text = stxt;
			},
			/**
			* 检测经纬度是否都有效
			* @param row{object}: 列表行数据
			**/
			checkLatLng(row) {
				var vm = this,
					lat = row.latitude,
					lng = row.longitude,
					valid = false;

				if(lat !== 'null' && lat !== '' && lat !== null && lng !== 'null' && lng !== '' && lng !== null) {
					valid = true;
				}

				return valid;
			},
			/**
			* 设备组选择项变动回调
			* @param selection{array}: 当前选择项
			**/
			groupChange(selection) {
				var vm = this;
				
				vm.groupNames = (selection||[]).map(function(item){ return item.group_name;}).join(', ');
				var ids = (selection||[]).map(function(item){ return item.id;});
				ids = ids.join(',');

				vm.groupIds = ids;
				// 刷新topo和统计信息数据
				vm.getTopoInfos(ids);
			},
			refreshTopoData() {
				var vm = this;

				if(vm.kpiMode === true) {
					vm.getKpiNodeData();
				}else {
					vm.getTopoInfos(vm.groupIds);
				}
			},
			// 状态控制设置
			setStatus() {
				var vm = this;

				Object.assign(vm.form, vm.statusForm);
				//vm.cancelSet();
			},
			// 取消设置关闭浮层
			cancelSet() {
				document.querySelector('#map').click();
				document.querySelector('#map').click();
			},
			nodeHasStatus(node) {
				var vm = this,
					deviceStatus = vm.statusForm.deviceStatus||[],
					activeStatus = vm.statusForm.activeStatus||[],
					sasStatus = vm.statusForm.sasStatus||[],
					hasStatus = false;

				// 设备类型和状态过滤
				if(deviceStatus.length) {
					deviceStatus.includes(node.online) && (hasStatus=true);
				}else if(activeStatus.length) {
					activeStatus.includes(node.active) && (hasStatus=true);
				}else if(sasStatus.length) {
					sasStatus.includes(node.sasState) && (hasStatus=true);
				}
				
				return hasStatus;
			},
			/**
			* 获取设备节点数据绘制topo
			* @param ids{string}: 设备组Id，多个以逗号分隔
			**/
			getTopoInfos(ids) {
				var vm = this,
					url = '${ctx}/cell/topo/getDeviceInfoList.action',
					deviceType = vm.statusForm.deviceType.join(',').toUpperCase(),
					params = {
						queryType: 'groupid',
						device_group_id: ids,
						device_type: deviceType,
						operator_code: operator_code
					};
				vm.links = {};
				// 获取enb节点
				
				vm.enbNodes = [];
				vm.gnbNodes = [];
				vm.gsmNodes = [];
				vm.cpeNodes = [];
				vm.allNodes = [];

				axios.post(url,stringify(params)).then(function(res){
					// 节点数据格式规范化 -- 应对变化的接口数据格式
					var data = res.data || {},
						enbNodes = data.ENB || [],
						gnbNodes = data.GNB || [],
						gsmNodes = data.GSM || [],
						cpeNodes = data.CPE || [];

					vm.enbNodes = transformNode(enbNodes);
					vm.gnbNodes = transformGnbNode(gnbNodes);
					vm.gsmNodes = transformNode(gsmNodes);
					vm.cpeNodes = vm.transformCPE(cpeNodes);

					vm.initMap();
				});
			},
			/**
			* 将获取的cpe节点数据转换为规范格式
			* @param nodes{array}: cpe原始数据
			**/
			transformCPE(nodes) {
				var vm = this,
					cpes = [],
					latLons = nodes.map(function(node){ return node.latitude+'-'+node.longitude; });
				
				nodes.map(function(node){
					var lat = node.latitude,
						lon = node.longitude;
					
					if(isNaN(node.latitude) || [null,undefined].includes(node.latitude)) lat = '';
					if(isNaN(node.longitude) || [null,undefined].includes(node.longitude))  lon = '';
					
					// 规范格式
					var cpe = {
							code: node.cpe_code,
							lat: lat,
							lon: lon,
							name: node.host_name,
							olat: node.latitude,
							olon: node.longitude,
							groupName: node.group_name,
							type: 'cpe',
							imsi: node.imsi,
							online: node.connection_status? 'on':'off',
							alarmCount: node.alarm_count,
							alarmLevel: node.alarm_serverity,
							ip: node.ipaddress,
							mac: node.macaddress,
							cellCode: node.cpe_code,
							sn: node.serial_number,
							alarm: node.alarm,
							sasEnable: node.sas_enable,
							sasState: node.sas_state,
							relaEnb: node.rela_enb,
							RSRP0: node.RSRP0,
							RSRP1: node.RSRP1,
							height: node.height
						};
					
					cpes.push(cpe);

					// 连线关系
					if(node.rela_enb) {
						var key = node.rela_enb,
							cpeCode = node.cpe_code;

						if(vm.links[key]) {
							vm.links[key].push(cpeCode);
						}else {
							vm.links[key] = [cpeCode]
						}
					}
				});
			
				return cpes;
			},
			/**
			* 设备组展开箭头点击事件
			* @param row{object}: 设备组列表行数据
			**/
			rowClick(row) {
				var vm = this;
				// 刷新设备列表数据
				vm.enbQueryParams.device_group_id = row.id;
				vm.gnbQueryParams.device_group_id = row.id;
				vm.gsmQueryParams.device_group_id = row.id;
				vm.cpeQueryParams.device_group_id = row.id;
				vm.$nextTick(function(){
					vm.deviceURL = '${ctx}/cell/topo/getDeviceInfoList.action';
					vm.cpeDeviceURL = '${ctx}/cell/topo/getDeviceInfoList.action';
				});
			},
			/**
			* 设备列表更多操作点击事件
			* @param row{object}: 设备列表行数据
			* @param ev{event}: 鼠标事件
			**/
			optClick(row,ev){
    	    	var vm = this,
					status = row.status,
    	    		isStopShow = status == 'on',
					cellCode = row.small_cell_code||row.cpe_code,
					serialNumber = row.serial_number;
			
				vm.gpsDeviceType = row.stationType?row.stationType.toLowerCase() : 'cpe';
				vm.curRow = row;
		    	vm.gpsDlg({
					lat: row.latitude,
					lon: row.longitude,
					cellCode: cellCode,
					sn: row.serial_number,
					height: row.height,
					mechanical_downtilt: row.mechanical_downtilt,
					vertical_3dB_beam_width: row.vertical_3dB_beam_width,
					horizontal_azimuth: row.horizontal_azimuth,
				});
    	    },
			modifyInfoGPS() {
				var vm = this,
					form = vm.infoForm;
				
				vm.gpsDeviceType = form.type || 'enb';
				vm.gpsDlg({
					lat: form.lat ? form.lat : '',
					lon: form.lon ? form.lon : '',
					cellCode: ['enb','gsm','gnb'].includes(vm.gpsDeviceType)?form.cellCode:form.code,
					sn: ['enb','gsm','gnb'].includes(vm.gpsDeviceType)?form.code:form.sn,
					height: form.height ? form.height : '',
					mechanical_downtilt: form.mechanical_downtilt,
					vertical_3dB_beam_width: form.vertical_3dB_beam_width,
					horizontal_azimuth: form.horizontal_azimuth,
				});
			},
			/**
			* 菜单点击事件
			* @param row{object}: 设备列表行数据
			**/
			gpsDlg(row) {
				var vm = this;
				
				vm.gpsdgVisible = true;
				vm.curDeviceCode = row.cellCode;
				vm.curSerialNumber = row.sn;
				vm.curMacAddress = row.sn;
				vm.$nextTick(function(){
					vm.$refs.gpsform.resetFields();
					Object.assign(vm.gpsForm,{
						lat: row.lat ? row.lat : '',
						lon: row.lon ? row.lon : '',
						height: row.height ? row.height : '',
						mechanical_downtilt: row.mechanical_downtilt,
						vertical_3dB_beam_width: row.vertical_3dB_beam_width,
						horizontal_azimuth: row.horizontal_azimuth,
					});
				})
			},
			// 设置经纬度
			setGPS() {
				var vm = this,
					url = '${ctx}/cell/topo/setLocationInfo.action',
					params = {
						cell_code: vm.curDeviceCode,
						longitude: vm.gpsForm.lon ? vm.gpsForm.lon : '',
						latitude: vm.gpsForm.lat ? vm.gpsForm.lat : '',
						setType: 'topo',
						operator_code: operator_code,
						height: vm.gpsForm.height ? vm.gpsForm.height : '',
						mechanical_downtilt: vm.gpsForm.mechanical_downtilt,
						vertical_3dB_beam_width: vm.gpsForm.vertical_3dB_beam_width,
						horizontal_azimuth: vm.gpsForm.horizontal_azimuth,
					},
					refMap = {
						'enb': vm.$refs.enbDevice,
						'gnb': vm.$refs.gnbDevice,
						'gsm': vm.$refs.gsmDevice,
						'cpe': vm.$refs.cpeDevice,
					},
					deviceType = vm.gpsDeviceType;
				// 保存设备经纬度信息
				vm.$refs.gpsform.validate(function(r){
					if(r) {
						axios.post(url,stringify(params)).then(function(res){
							if(res.data['success']){
								vm.gpsdgVisible = false;

								if(refMap[deviceType]) {
									// 刷新设备列表数据
									refMap[deviceType].refresh();
								}

								var codes = vm.allNodes.map(function(item){return item.cellCode;});
								if(codes.includes(vm.curDeviceCode)) {
									
									if(vm.curDeviceCode == vm.infoForm.cellCode) {
										Object.assign(vm.infoForm, {
											lon: vm.gpsForm.lon,
											lat: vm.gpsForm.lat,
											height: vm.gpsForm.height,
											mechanical_downtilt: vm.gpsForm.mechanical_downtilt,
											vertical_3dB_beam_width: vm.gpsForm.vertical_3dB_beam_width,
											horizontal_azimuth: vm.gpsForm.horizontal_azimuth,
										})
									}

									var ids = vm.$refs.group.getChecked().join(',');
									vm.getTopoInfos(ids);
								}else {
									var ids = vm.$refs.group.getChecked().join(',');
									vm.getTopoInfos(ids);
								}
							}else{
								toast(res.data['message'],$('#omc_app_ctn'));
							}
						})
					}
				})
			},
			/**
			* 获取topo可视边界域
			* @param resObj{object}: 所有节点的统计数据对象
			**/
			getViewBounds(resObj) {

				return [
					[resObj.latmin, resObj.lonmin], // 西南角点
					[resObj.latmax, resObj.lonmax]  // 东北角点
				];
			},
			/**
			* 根据设置面板过滤节点
			* @param nodes{array}: 所有节点
			**/
			filterNodesByStatus(nodes) {
				var vm = this,
					deviceType = vm.statusForm.deviceType||[]
					deviceStatus = vm.statusForm.deviceStatus||[],
					activeStatus = vm.statusForm.activeStatus||[],
					sasStatus = vm.statusForm.sasStatus||[];

				// 设备类型和状态过滤
				if(deviceStatus.length) {
					nodes = nodes.filter(function(node){
						var online = node.online;

						return deviceStatus.includes(online);
					});
				}else if(activeStatus.length) {
					nodes = nodes.filter(function(node){
						var active = node.active;

						return activeStatus.includes(active);
					});
				}else if(sasStatus.length) {
					nodes = nodes.filter(function(node){
						var sas = node.sasState;

						return sasStatus.includes(sas);
					});
				}
				// 如果没有任何状态过滤器，不过滤节点（保留所有节点）
				// 注意：之前这里会返回空数组，导致没有节点显示
				
				nodes = nodes.filter(function(node){
					// 使用 getNodeType 函数统一获取节点类型，支持 enb/gnb/gsm/cpe
					var type = vm.getNodeType(node);

					return deviceType.includes(type);
				});
				
				return nodes;
			},
			/**
			* OpenLayers 版本：地图缩放或拖拽后的节点过滤和渲染
			* @param nodes{array}: 所有要绘制的节点集
			* @param olHelper{OLTopoHelper}: OpenLayers 辅助类实例
			* @param bufEnable{boolean}: 是否开启缓存
			**/
			filterRenderOL(nodes, olHelper, bufEnable) {
				var vm = this,
					threshold = this.limit || 5000;
				
				// 根据控制状态过滤节点
				nodes = vm.filterNodesByStatus(nodes);
				
				// 使用 OpenLayers Helper 的高性能过滤渲染
				olHelper.filterVisibleNodes(nodes, threshold);
			},
			/**
			* 显示节点悬浮信息
			* @param nodeData{object}: 节点数据
			* @param evt{event}: 事件对象
			**/
			showNodeHoverInfo(nodeData, evt, olHelper, feature) {
				var vm = this;
				
				if (!nodeData || !olHelper || !olHelper.overlay) {
					return;
				}

				// 构建悬浮提示框内容
				var statusText = '';
				var statusClass = '';
				var activeText = '';
				var activeClass = '';
				// 设备状态
				if (nodeData.online === 'on') {
					statusText = '<i class="el-icon el-icon-status-conn-on margin-right-5"></i><%=rb.getString("ZaiXian")%>';
					statusClass = 'status-online';
				} else {
					statusText = '<i class="el-icon el-icon-status-conn-off margin-right-5"></i><%=rb.getString("LiXian")%>';
					statusClass = 'status-offline';
				}
				// 小区激活状态
				if (nodeData.active === 'yes') {
					activeText = '<i class="el-icon el-icon-status-active online margin-right-5"></i></i><%=rb.getString("JiHuo")%>';
					activeClass = 'status-online';
				} else {
					activeText = '<i class="el-icon el-icon-status-active inactive margin-right-5"></i><%=rb.getString("QuJiHuo")%>';
					activeClass = 'status-offline';
				}
				var nodeTypeText = nodeData.type === 'enb' ? 'ENB' : (nodeData.type === 'gnb' ? 'GNB' : (nodeData.type === 'gsm'? 'GSM':'CPE'));

				// 构建HTML内容
				var htmlContent = '<div class="popup-content">';
				htmlContent += '<a href="#" class="ol-popup-closer" onclick="event.preventDefault(); window.globOLHelper && window.globOLHelper.hideOverlay();">×</a>';
				htmlContent += '<h3>' + nodeTypeText + ' Information</h3>';
				
				htmlContent += '<div class="info-row">';
				htmlContent += '<span class="info-label"><%=rb.getString("DianYuanBianMa")%>:</span>';
				htmlContent += '<span class="info-value">' + (nodeData.code || nodeData.sn || 'N/A') + '</span>';
				htmlContent += '</div>';
				// 设备状态
				htmlContent += '<div class="info-row">';
				htmlContent += '<span class="info-label"><%=rb.getString("SheBeiZhuangTai")%>:</span>';
				htmlContent += '<span class="info-value ' + statusClass + '">' + statusText + '</span>';
				htmlContent += '</div>';
				
				
				if(['enb', 'gsm', 'gnb'].includes(nodeData.type)) {
					// 小区激活状态
					htmlContent += '<div class="info-row">';
					htmlContent += '<span class="info-label"><%=rb.getString("ShiFouJiHuo")%>:</span>';
					htmlContent += '<span class="info-value ' + activeClass + '">' + activeText + '</span>';
					htmlContent += '</div>';
					// 告警级别
					var alarmText = alarmPopoverFormatter(nodeData.alarm, nodeData);
					htmlContent += '<div class="info-row">';
					htmlContent += '<span class="info-label"><%=rb.getString("GaoJingJiBie")%>:</span>';
					htmlContent += '<span class="info-value">' + alarmText + '</span>';
					htmlContent += '</div>';
				}
				// SAS 状态
				if(writableMap['CODE_ADVANCE_SAS']!=undefined) {
					var sasText = sasStatusFormatter(nodeData.sasState, nodeData);
					htmlContent += '<div class="info-row">';
					htmlContent += '<span class="info-label"><%=rb.getString("SASZhuangTai")%>:</span>';
					htmlContent += '<span class="info-value">' + sasText + '</span>';
					htmlContent += '</div>';
				}

				// UE | MDT 数据
				if (nodeData.type === 'enb' && vm.statusForm.ueStatus.includes('mdt') ) {
					htmlContent += '<div class="info-row">';
					htmlContent += '<span class="info-label">MDT:</span>';
					htmlContent += '<span class="info-value">' + 
										'<span class="ue-count" code="' + nodeData.code + '" >' +
											(nodeData.hasMDT ? '<i class="el-icon el-icon-MDT mdt-flag" onclick="showUEcount(this,event)"></i>' : '') +
										'</span>' +
									'</span>';
					htmlContent += '</div>';
				}
				
				// CPE 特有信息
				if (nodeData.type === 'cpe') {
					htmlContent += '<div class="info-row">';
					htmlContent += '<span class="info-label">eNB SN:</span>';
					htmlContent += '<span class="info-value">' + nodeData.relaEnb + ' dBm</span>';
					htmlContent += '</div>';
				}
				
				htmlContent += '</div>';

				// 显示overlay
				// 如果是展开的节点，使用feature的实际坐标；否则使用nodeData中的经纬度
				var coordinate;
				if (feature && feature.get('isExpanded')) {
					// 展开节点：从feature的几何体获取实际坐标，然后转换为经纬度
					var featureCoord = feature.getGeometry().getCoordinates();
					var lonLat = ol.proj.toLonLat(featureCoord);
					coordinate = lonLat;
				} else {
					// 普通节点：使用nodeData中的经纬度
					coordinate = [parseFloat(nodeData.lon), parseFloat(nodeData.lat)];
				}
				
				var popupElement = olHelper.showOverlay(coordinate, nodeData);
				if (popupElement) {
					popupElement.innerHTML = htmlContent;
					popupElement.style.display = 'block';
				}
			},
			/* 隐藏节点悬浮信息 */
			hideNodeHoverInfo() {
				var vm = this;
				// 隐藏悬浮提示
				if (window.globOLHelper && window.globOLHelper.overlay) {
					window.globOLHelper.hideOverlay();
					var popupElement = window.globOLHelper.overlay.getElement();
					if (popupElement) {
						popupElement.style.display = 'none';
					}
				}
			},
			/**
			* 显示节点详细信息（点击后）
			* @param nodeData{object}: 节点数据
			**/
			showNodeDetail(nodeData) {
				var vm = this;
				// 显示右侧或左侧详情面板
				vm.infoShow = true;
				// 填充详情数据
				Object.assign(vm.infoForm, {
					type: nodeData.type,
					cellCode: nodeData.cellCode,
					code: nodeData.code || nodeData.sn,
					sn: nodeData.code || nodeData.sn,
					name: nodeData.name || nodeData.host_name,
					lat: nodeData.lat,
					lon: nodeData.lon,
					status: nodeData.status,
					active: nodeData.active,
					alarm: nodeData.alarm,
					mmeStatus: nodeData.mmeStatus,
					NEW_MME_STATUS: nodeData.NEW_MME_STATUS,
					sasEnable: ['1','on'].includes(nodeData.sasEnable)?'on':'off',
					sasState: nodeData.sasState,
					ueCount: nodeData.ueCount,
					pci: nodeData.pci,
					radius: nodeData.radius,
					minRadius: nodeData.minRadius,
					// CPE 特有字段
					imsi: nodeData.imsi,
					ip: nodeData.ip,
					mac: nodeData.mac,
					relaEnb: nodeData.relaEnb,
					groupName: nodeData.groupName,
					rsrp: nodeData.rsrp,
					// GPS 定位相关字段
					height: nodeData.height,
					mechanical_downtilt: nodeData.mechanical_downtilt,
					vertical_3dB_beam_width: nodeData.vertical_3dB_beam_width,
					horizontal_azimuth: nodeData.horizontal_azimuth
				});
			},
			// 初始化topo图
			async initMap() {
				var vm = this,
					eNodebs = vm.enbNodes.concat(vm.gnbNodes).concat(vm.gsmNodes).concat(vm.cpeNodes);
				
				vm.closeMesure();

				// 如果地图已初始化过，保存当前的视图状态（中心点和缩放级别）
				if (vm.isMapInitialized && globMap && globMap.getView) {
					var view = globMap.getView();
					vm.savedMapView = {
						center: view.getCenter(),
						zoom: view.getZoom()
					};
				}

				// 销毁旧地图实例
				if(globMap) {
					if(globMap.destroy) {
						globMap.destroy();
					} else if(globMap.setTarget) {
						globMap.setTarget(null);
					}
				}
				
				// 处理基站数据格式
				var resObj = vm.proccessNodes(eNodebs);
				// 获取要展示的区域边界 [minLat, minLon, maxLat, maxLon]
				var bounds = [
					resObj.latmin || -90,
					resObj.lonmin || -180,
					resObj.latmax || 90,
					resObj.lonmax || 180
				];

				// 创建 OpenLayers 辅助类实例
				if(!window.globOLHelper) {
					window.globOLHelper = new OLTopoHelper();
				}
				
				var olHelper = window.globOLHelper;
				
				// 首次初始化时，先预加载所有SVG图标
				if (!vm.isMapInitialized) {
					await olHelper.preloadAllSvgs();
				}
				
				// 初始化地图
				var mapOptions = {
					containerId: 'map',
					bounds: bounds,
					offlineMapEnable: offlineMapEnable,
					mapUrl: offlineMapEnable ? '${ctx}/map/{z}/{x}/{y}.png' : null
				};
				
				globMap = olHelper.initMap(mapOptions);
				
				// 如果不是首次初始化且有保存的视图状态，则恢复视图
				if (vm.isMapInitialized && vm.savedMapView && globMap && globMap.getView) {
					var view = globMap.getView();
					view.setCenter(vm.savedMapView.center);
					view.setZoom(vm.savedMapView.zoom);
				} else {
					// 首次初始化，标记为已初始化
					vm.isMapInitialized = true;
				}
				
				// 自适应窗口设置（移到 globOLHelper 初始化之后）
				var $map = $('#map');
				var resize = function () {
					$map.height($('.topo-ctn').height() - 20);
					if (window.globOLHelper && window.globOLHelper.map) {
						window.globOLHelper.map.updateSize();
					}
				};
				$(window).off('resize').on('resize', resize);
				resize();
				
				// 过滤并创建节点图层
				var filteredNodes = vm.filterNodesByStatus(resObj.enb);
				vm.allNodes = resObj.enb;
				
				// 创建高性能节点图层
				olHelper.createNodesLayer(filteredNodes);
				
				// 创建 CPE 到 ENB 的连线图层
				var cpeNodesWithRelaEnb = filteredNodes.filter(function(node) {
					return node.type === 'cpe' && node.relaEnb;
				});
				if (cpeNodesWithRelaEnb.length > 0) {
					olHelper.createLinesLayer(cpeNodesWithRelaEnb, resObj.enb);
				}
				
				// 创建悬浮提示框（如果需要）
				var popupElement = document.getElementById('ol-popup');
				if (!popupElement) {
					popupElement = document.createElement('div');
					popupElement.id = 'ol-popup';
					popupElement.className = 'ol-popup';
					popupElement.style.display = 'none';
					document.body.appendChild(popupElement);
				}
				olHelper.createOverlay(popupElement);

				// 添加鼠标悬浮交互
				olHelper.addHoverInteraction(function(nodeData, evt, feature) {
					if (nodeData) {
						// 显示悬浮信息
						vm.showNodeHoverInfo(nodeData, evt, olHelper, feature);
					} else {
						// 隐藏悬浮信息
						vm.hideNodeHoverInfo();
					}
				});

				// 地图初始化完成后，重新初始化围栏系统
				vm.$nextTick(function() {
					// 添加围栏点击监听器
					if (window.globOLHelper && window.globOLHelper.map) {
						window.globOLHelper.addFenceClickListener(function(fenceParams) {
							vm.showFenceDevices(fenceParams);
						});
						// 清除所有旧的围栏，避免重复叠加
						window.globOLHelper.clearAllFences();
						// 如果有保存的围栏数据，自动显示在地图上
						if (vm.fenceEnable && vm.fenceList && vm.fenceList.length > 0) {
							vm.fenceList.forEach(function(fenceParams) {
								try {
									if(fenceParams.enable === true) {
										window.globOLHelper.drawFenceFromParams(fenceParams);
									}
								} catch (e) {}
							});
						}
					}
				});
				
				// 初始化节点拖拽功能（基于sasEnable）
				olHelper.enableNodeDrag(!vm.sasEnable, function(feature, originalLonLat, newLonLat, originalCoords) {
					// 获取节点数据
					var nodeData = feature.get('nodeData');
					if (!nodeData) return;
					
					// 根据节点类型获取正确的cell_code
					var cellCode;
					if (nodeData.type === 'cpe') {
						// CPE节点使用sn
						cellCode = nodeData.sn;
					} else {
						// eNB/gNB/GSM节点使用code
						cellCode = nodeData.cellCode;
					}
					
					// 设置gpsmvoeForm
					vm.gpsmvoeForm.lat = newLonLat[1].toFixed(6);
					vm.gpsmvoeForm.lon = newLonLat[0].toFixed(6);
					vm.gpsmvoeForm.olatlng = {
						lat: originalLonLat[1],
						lon: originalLonLat[0]
					};
					vm.gpsmvoeForm.cell_code = cellCode;
					vm.gpsmvoeForm.target = feature;
					vm.gpsmvoeForm.originalPosition = originalCoords; // 显示确认对话框
					vm.gpsmoveVisible = true;
				});
				
				// 添加点击交互
				olHelper.addClickInteraction(function(nodeData, evt, feature) {
					if (feature) {
						// 检查是否是展开状态的节点
						var isExpanded = feature.get('isExpanded');
						
						if (isExpanded) {
							// 点击的是展开后的节点，显示详细信息
							vm.showNodeDetail(nodeData);
							// 高亮显示点击的节点
							if (nodeData) {
								var coordinate = feature.getGeometry().getCoordinates();
								olHelper.highlightNode(nodeData, coordinate);
							}
							return;
						}
						
						// 检查是否是重叠节点
						var isCluster = feature.get('isCluster');
						var allNodes = feature.get('allNodes');
						
						if (isCluster && allNodes && allNodes.length > 1) {
							// 如果是重叠节点，扇形展开
							olHelper.expandCluster(feature);
							// 同时显示右侧面板
							vm.showClusterNodes(allNodes);
						} else if (nodeData) {
							// 单个节点，检查是否有扇面数据
							if (olHelper.hasRequiredSectorData(nodeData)) {
								// 有数据时，绘制小区扇面
								olHelper.drawCellSector(nodeData);
							}
							// 显示详细信息面板
							vm.showNodeDetail(nodeData);
							// 高亮显示点击的节点
							olHelper.highlightNode(nodeData);
						}
					} else {
						olHelper.collapseCluster(true);
					}
				});				
				// 添加缩放和移动事件监听（用于性能优化）
				olHelper.addViewChangeListeners(function(eventType) {
					// 收起展开的cluster（地图操作时），但尊重keepExpandedOnViewChange标志
					// 不传force参数，这样如果用户手动展开了cluster，就不会自动收起
					if (eventType === 'zoom') {
						// 缩放时总是收起，因为缩放会改变节点位置的视觉效果
						olHelper.collapseCluster();
						// 缩放结束后的处理
						vm.filterRenderOL(resObj.enb, olHelper);
					} else if (eventType === 'move') {
						// 移动结束后的处理
						vm.filterRenderOL(resObj.enb, olHelper, true);
					}
				});
			},
			closeMesure() {
				// 如果使用OpenLayers地图，调用其stopMeasure方法
				if (window.globOLHelper && window.globOLHelper.measureActive) {
					window.globOLHelper.stopMeasure();
					$('.flex-bt-cls').removeClass('selected');
				} 
			},
			/**
			* 计算经纬边界值
			* @param nodes{array}: 所有节点
			**/
			proccessNodes(nodes) {
				var cLat = 0, cLon = 0, 
					latMax = -90, lonMax = -180,
					latMin = 90, lonMin = 180;
				
				nodes.map(function(item,index){
					var ilat = parseFloat(item.lat),
						ilon = parseFloat(item.lon);
					if(ilat>latMax && Math.abs(ilat)<=90) latMax = ilat;
					if(ilon>lonMax && Math.abs(ilon)<=180) lonMax = ilon;
					if(ilat<latMin && Math.abs(ilat)<=90) latMin = ilat;
					if(ilon<lonMin && Math.abs(ilon)<=180) lonMin = ilon;
					
				});
				
				// 计算经纬中心点
				cLat = (latMax + latMin)/2;
				cLon = (lonMax + lonMin)/2;

				return {
					enb: nodes,
					latMax: latMax,
					lat: cLat,
					lon: cLon,
					latdis: latMax - latMin,
					londis: lonMax - lonMin,
					latmax: latMax,
					latmin: latMin,
					lonmax: lonMax,
					lonmin: lonMin
				}
			},
			/**
			* 获取连线数据
			* @param nodes{array}: 所有节点
			**/
			proccessLines(nodes) {
				var vm = this,
					lineList = [],
					enblist = nodes.filter(function(item){ return item.type=='enb';}),
					cpelist = nodes.filter(function(item){ return item.type=='cpe';});

				enblist.map(function(node){
					var cpekeys = vm.links[node.code];
					// 存在连线
					if(cpekeys && cpekeys.length) {
						cpekeys.map(function(cpekey){
							var cpe = cpelist.filter(function(item){
									return item.code == cpekey && item.lat!==null && item.lon!==null && item.lat!=="" && item.lon!==""; // 根据cpe key获取cpe数据
								})[0];
							
							if(cpe) {
								var cpeLine = {
										"label": 'CPE',
										"enb": node.code,
										"tar": cpe.code,
										"cnt": "",
										type: 'enb',
										online: node.online,
										active: node.active,
										mmeStatus: node.mmeStatus,
										mmeEnable: node.mmeEnable
									};

								lineList.push(cpeLine);
							}
						})
					}
				});

				return lineList;
			},
			/**
			* 设备列表滑动检测
			* @param ev{event}: 鼠标事件
			**/
			observSlider(ev){// 通过事件代理，将逻辑抽离到父级，避免动态节点时，子节点的复杂绑定逻辑
				var vm = this,
					target = ev.target,
					clsList = target.classList;

				/* 切换当前行箭头状态 */
				if(clsList.contains('el-icon-arrow-right') || clsList.contains('el-icon-arrow-left')) {
					if(clsList.contains('el-icon-arrow-right')) {
						vm.showSlider(ev,true);
					}else {
						vm.showSlider(ev,false);
					}

					['el-icon-arrow-right','el-icon-arrow-left'].map(function(cls){
						if(clsList.contains(cls)) clsList.remove(cls);
						else clsList.add(cls);
					});
					/* 重置其他行箭头状态 */
					Array.from(document.querySelectorAll('.row-op')).map(function(item){
						if(item != target) {
							item.classList.remove('el-icon-arrow-left');
							item.classList.add('el-icon-arrow-right');
						}
					});
				}
			},
			/**
			* 设备列表滑动检测
			* @param evt{event}: 鼠标事件
			* @param bool{boolean}: 是否展开
			**/
			showSlider(evt,bool) {
				var vm = this,
					slider = vm.$refs.deviceSlider;
				
				/*根据事件对象调整位置*/
				if(evt) {
					var pos = vm.reposition(slider,evt.target),
						top = pos.top;
				}

				slider.classList.add('loading');
				if(bool) slider.classList.add('show');
				else slider.classList.remove('show');

				setTimeout(function(){
					slider.classList.remove('loading');
				},1000);

				vm.$refs.enblist.advanceQuery();
				vm.enbQueryParams.search_text = '';
				vm.gnbQueryParams.search_text = '';
				vm.gsmQueryParams.search_text = '';
				vm.cpeQueryParams.search_text = '';
			},
			/* *
			 * 相对偏移值  -- 实现动态计算、精准移动目标
			 * @param target: 计算的目标对象
			 * @param reference: 计算的参照物
			 * */
			reposition(target, reference) {
				var docH = window.innerHeight || document.documentElement.clientHeight ||
					document.body.clientHeight,
					tRect = reference.getBoundingClientRect(),
					mRect = target.getBoundingClientRect();
				/**
				 * 根据相对视口的坐标偏移，计算相对位置的偏移量
				 *（top偏移：rect的top的坐标偏移差，left偏移：rect的left坐标偏移差）
				 **/
				var tTop = tRect.top,
					tLeft = tRect.left,
					mTop = mRect.top,
					mLeft = mRect.left,
					top = target.offsetTop + (tTop - mTop) - 10,
					left = target.offsetLeft + (tLeft - mLeft);
				var offsetD = tRect.y + tRect.height;

				if (docH - offsetD < mRect.height) {
					//top = top - mRect.height - tRect.height;
					top -= mRect.height - docH + offsetD + 30;
				}
				return {
					left: left,
					top: top
				};
			},
			// 全屏设置
			FullScreen() {
				var vm = this;
				var el = document.querySelector('#mainpage');

				if(navTabsEnable === true) {
					el = document.querySelector('#topo_container');
				}

				var isFullscreen = document.fullScreen || document.mozFullScreen || document.webkitIsFullScreen;
				if (!isFullscreen) { //进入全屏,多重短路表达式
					// 保存将要进入全屏的元素引用
					vm._fullscreenElement = el;
					
					(el.requestFullscreen && el.requestFullscreen()) ||
					(el.mozRequestFullScreen && el.mozRequestFullScreen()) ||
					(el.webkitRequestFullscreen && el.webkitRequestFullscreen()) || 
					(el.msRequestFullscreen && el.msRequestFullscreen());
					
					// 监听全屏成功事件，将现有和未来的弹窗移动到全屏容器内
					setTimeout(function() {
						vm.moveDialogsToFullscreenContainer();
						vm.observeDialogCreation();
					}, 100);

				} else { //退出全屏,三目运算符
					// 先将所有弹窗移回 body
					vm.restoreDialogsToBody();
					
					document.exitFullscreen ? document.exitFullscreen() : document.mozCancelFullScreen ? document.mozCancelFullScreen() : document.webkitExitFullscreen ? document.webkitExitFullscreen() : '';
					// 清理观察器
					if(vm._dialogObserver) {
						vm._dialogObserver.disconnect();
						vm._dialogObserver = null;
					}
				}
			},
			/* 将所有弹窗恢复到 body */
			restoreDialogsToBody() {
                var vm = this;
				// 在退出全屏时，先获取当前全屏元素（如果有）
				// 或者使用保存的全屏元素引用
				var fullscreenEl = vm._fullscreenElement || 
				                   document.fullscreenElement || 
				                   document.mozFullScreenElement || 
				                   document.webkitFullscreenElement || 
				                   document.msFullscreenElement;
				
				if (!fullscreenEl) return;
				
				// 将所有从 body 移动到全屏容器的元素移回 body
				var selectors = [
					'.messager-body',
					'.messager-window',
					'.el-dialog__wrapper',
					'.v-modal',
					'.window-mask',
					'.el-select-dropdown',
					'.el-picker-panel',
					'.el-cascader-menus',
					'.el-autocomplete-suggestion',
					'.el-popover',
					'.el-tooltip__popper',
					'.el-color-picker__panel',
					'.combo-p',
					'.combo-panel',
					'.el-message-box__wrapper',
					'.el-message',
					'.el-notification'
				];
				
				selectors.forEach(function(selector) {
					var elements = fullscreenEl.querySelectorAll(selector);
					Array.from(elements).forEach(function(element) {
						// 移回 body
						if (element.parentElement === fullscreenEl) {
							document.body.appendChild(element);
							
							// 恢复保存的位置信息（如果有）
							if (element._savedPosition) {
								element.style.top = element._savedPosition.top;
								element.style.left = element._savedPosition.left;
								element.style.position = element._savedPosition.position;
								delete element._savedPosition;
							}
						}
					});
				});
				
				// 清除保存的全屏元素引用
				vm._fullscreenElement = null;
			},
			/* 将所有弹窗移动到全屏容器内 */
			moveDialogsToFullscreenContainer() {
				var fullscreenEl = document.fullscreenElement || document.mozFullScreenElement ||  document.webkitFullscreenElement ||  document.msFullscreenElement;
				
				if (!fullscreenEl) return;
				
				// 移动 jQuery EasyUI Messager 弹窗
				var messagerDialogs = document.querySelectorAll('body > .messager-body, body > .messager-window');
				Array.from(messagerDialogs).forEach(function(dialog) {
					if (dialog.parentElement === document.body) {
						// 保存弹窗当前的位置信息（用于恢复）
						var computedStyle = window.getComputedStyle(dialog);
						dialog._savedPosition = {
							top: dialog.style.top || computedStyle.top,
							left: dialog.style.left || computedStyle.left,
							position: dialog.style.position || computedStyle.position
						};
						
						fullscreenEl.appendChild(dialog);
					}
				});
				// 移动 Element UI Dialog 弹窗
				var elDialogs = document.querySelectorAll('body > .el-dialog__wrapper');
				Array.from(elDialogs).forEach(function(dialog) {
					if (dialog.parentElement === document.body) {
						// 保存弹窗当前的位置信息
						var computedStyle = window.getComputedStyle(dialog);
						dialog._savedPosition = {
							top: dialog.style.top || computedStyle.top,
							left: dialog.style.left || computedStyle.left,
							position: dialog.style.position || computedStyle.position
						};
						
						fullscreenEl.appendChild(dialog);
					}
				});
				// 移动 Element UI 遮罩层
				var modals = document.querySelectorAll('body > .v-modal');
				Array.from(modals).forEach(function(modal) {
					if (modal.parentElement === document.body) {
						fullscreenEl.appendChild(modal);
					}
				});
				// 移动 jQuery EasyUI 遮罩层
				var windowMasks = document.querySelectorAll('body > .window-mask');
				Array.from(windowMasks).forEach(function(mask) {
					if (mask.parentElement === document.body) {
						fullscreenEl.appendChild(mask);
					}
				});
				// 移动 Element UI 下拉框和选择器
				var selectors = 'body > .el-select-dropdown, body > .el-picker-panel, body > .el-cascader-menus, body > .el-autocomplete-suggestion, body > .el-popover, body > .el-tooltip__popper, body > .el-color-picker__panel, body > .combo-p, body > .combo-panel, body > .el-message-box__wrapper, body > .el-message, body > .el-notification';
				var dropdowns = document.querySelectorAll(selectors);
				Array.from(dropdowns).forEach(function(dropdown) {
					if (dropdown.parentElement === document.body) {
						fullscreenEl.appendChild(dropdown);
					}
				});
			},
			/* 观察新创建的弹窗并自动移动到全屏容器 */
			observeDialogCreation() {
				var vm = this;
				var fullscreenEl = document.fullscreenElement || document.mozFullScreenElement || document.webkitFullscreenElement ||  document.msFullscreenElement;
				
				if (!fullscreenEl || !window.MutationObserver) return;
				
				// 创建 MutationObserver 监听 body 的子节点变化
				vm._dialogObserver = new MutationObserver(function(mutations) {
					mutations.forEach(function(mutation) {
						Array.from(mutation.addedNodes).forEach(function(node) {
							if (node.nodeType === 1) { // 元素节点
								// 检查是否是弹窗、遮罩层或下拉框相关元素
								var isDialog = node.classList.contains('messager-body') ||  node.classList.contains('messager-window') || node.classList.contains('el-dialog__wrapper') || 
								              node.classList.contains('v-modal') || node.classList.contains('window-mask') ||
								              // Element UI 下拉框和选择器
								              node.classList.contains('el-select-dropdown') || node.classList.contains('el-picker-panel') || node.classList.contains('el-cascader-menus') ||
								              node.classList.contains('el-autocomplete-suggestion') || node.classList.contains('el-popover') ||
								              node.classList.contains('el-tooltip__popper') || node.classList.contains('el-color-picker__panel') ||
								              // jQuery EasyUI 下拉框 (添加 combo-p)
								              node.classList.contains('combo-p') || node.classList.contains('combo-panel') ||
								              // Element UI MessageBox/Message/Notification
								              node.classList.contains('el-message-box__wrapper') || node.classList.contains('el-message') || node.classList.contains('el-notification');
								
								if (isDialog && node.parentElement === document.body) {
									// 将新创建的弹窗移动到全屏容器
									var currentFullscreenEl = document.fullscreenElement || document.mozFullScreenElement || document.webkitFullscreenElement || document.msFullscreenElement;
									if (currentFullscreenEl) {
										currentFullscreenEl.appendChild(node);
									}
								}
							}
						});
					});
				});
				
				// 开始观察 body 的子节点变化
				vm._dialogObserver.observe(document.body, {
					childList: true,
					subtree: false
				});
				
			},
			hidePanel() {
				var vm = this;

				vm.infoShow = false;
			},
			resetSiteStatus() {
				var vm = this;

				vm.deviceDetailShow = false;
				vm.topolistShow = false;
				vm.siteModifyHidden = false;
				vm.addSiteDeviceShow = false;
				vm.siteModifyShow = false;
				vm.siteInfoShow = false;
				vm.siteAddHidden = false;
				vm.siteTab = 'deviceList';
				document.querySelector('#site_map').click();
			},
			addSite() {
				var vm = this;

				vm.siteType = 'add';
				vm.resetSiteStatus();
				vm.siteInfoShow = true;
				vm.siteAddHidden = false;
				Object.assign(vm.siteAddForm, {
					siteName: '',
					latitude: '',
					longitude: '',
					cellCodes: '',
					locationConsistent: false,
				});
				vm.siteAddDeviceType = vm.enbEnable? ['eNB']:(vm.gnbEnable? ['gNB']:(vm.isGSMEnable? ['GSM']:[]));
				vm.$refs.addTb.clearSelection();
			},
			siteOptClick(row, evt) {
				var vm = this,
					topoShow = false,
					isNotSBS7 = !['sBS72030','sBS71400'].includes(row.module_type);

				if(['eNB'].includes(row.stationType)) {
					var paramsCode={
							smallCellCode: row.small_cell_code
						},
						showFlagList = [];
		
					$.ajax({
						type: 'POST',
						url: '${ctx}/cell/cpeinfos/getENBOperationItem.action',
						data: paramsCode,
						async: false,
						dataType: 'json',
						success: function(data){
							var operationData = data;
							if(operationData.Maintenance && operationData.Maintenance.length > 0 ){
								operationData.Maintenance.map((item)=>{
									showFlagList[item] = true;
								});
							}
							if(operationData.Actions && operationData.Actions.length > 0 ){
								operationData.Actions.map((item)=>{
									showFlagList[item] = true;
								})
							}
							if(operationData.others && operationData.others.length > 0 ){
								operationData.others.map((item)=>{
									showFlagList[item] = true;
								})
							}
						},
					});

					if(showFlagList.CODE_ENB_DISTRIBUTED && isNotSBS7) {
						topoShow = true;
					}
				}else if(['gNB'].includes(row.stationType)) {
					if(row.product != 'BaiBNQ') {
						topoShow = true;
					}
				}else if(['GSM'].includes(row.stationType)) {
					topoShow = false;
				}

				vm.siteMenus = [
					{label: '<%=rb.getString("XinXi")%>', code: 'view', cls: 'el-icon el-icon-operation-details', row: row},
					{label: 'TOPO', code: 'topo', cls: 'el-icon el-icon-operation-topo', row: row, show: topoShow},
					{label: '<%=rb.getString("SheZhi")%>', code: 'setting', cls: 'el-icon el-icon-operation-settings', row: row},
					{label: '<%=rb.getString("ShanChu")%>', code: 'delete', cls: 'el-icon el-icon-operation-delete', row: row}
				];

				vm.$nextTick(function(){
					document.body.click();
					vm.$refs.siteMenu.show(evt);
				})
			},
			menuClick(row) {
				var vm = this,
					code = row.code,
					rowData = row.row,
					actions = {
						view: vm.viewSiteDevice,
						topo: vm.toTOPOList,
						setting: vm.modifySite,
						delete: vm.delSiteDevice,
					}

				if(typeof(actions[code]) == 'function') {
					vm.resetSiteStatus();
					vm.siteInfoShow = true;
					actions[code](rowData);
				}
			},
			handerClose() {
				this.$refs.siteMenu.hide();
			},
			querySite(txt) {
				var vm = this;

				vm.siteQuery.siteName = txt;
				vm.siteQuery.rd = Math.random();
			},
			siteCurrentChange(row, oldRow) {
				var vm = this;

				vm.siteType = 'view';
				vm.currentSiteRow = row;
				vm.viewQuery.siteName = row.siteName;
				vm.viewDeviceURL = '${ctx}/cell/topo/getDeviceInfoListForSite.action';
				vm.resetSiteStatus();
				vm.siteInfoShow = true;
				vm.siteAddHidden = false;
			},
			viewQueryClick(txt) {
				var vm = this,
					trimTxt = txt.trim();

				vm.viewQuery.search_text = trimTxt;
				vm.viewQuery.rd = Math.random();
			},
			addQueryClick(txt) {
				var vm = this,
					trimTxt = txt.trim();

				vm.addQuery.searchText = trimTxt;
			},
			deviceToSiteQueryClick(txt) {
				var vm = this,
					trimTxt = txt.trim();

				vm.deviceToSiteQuery.searchText = trimTxt;
			},
			addSelectionChange(s) {
				var vm = this,
					list = s.map(item=>{
						return item.small_cell_code;
					});

				vm.siteAddForm.cellCodes = list.join(',');
			},
			deviceToSiteSelectChange(s) {
				var vm = this,
					list = s.map(item => {
						return item.small_cell_code
					});

				vm.deviceToSiteForm.cellCodes = list.join(',');
			},
			addSiteSubmit() {
				var vm = this,
					url = '${ctx}/site/addSiteInfo.action',
					params = {};

				Object.assign(params, vm.siteAddForm);

				vm.$refs.addForm.validate(function(r){
					if(r) {
						axios.post(url, stringify(params)).then((res) => {
							var data = res.data;

							if(data.success == true) {
								vm.siteInfoShow = false;
								vm.$refs.siteList.refresh();
								vm.getSiteList();

								vm.$message({
									message: '<%=rb.getString("ChengGong")%>',
									type:'success'
								})
							}else {
								vm.$message.error(data["message"])
							}
						})
					}
				})
			},
			modifySite(row, evt) {
				var vm = this;

				if(row.serial_number) {
					Object.assign(vm.siteModifyForm,{
						serialNumber: row.serial_number,
						cellCode: row.small_cell_code,
						siteName: '',
						latitude: row.latitude,
						longitude: row.longitude
					})
				}else {
					Object.assign(vm.siteModifyForm,{
						serialNumber: '',
						cellCode: '',
						siteName: row.siteName,
						latitude: row.latitude,
						longitude: row.longitude
					})
				}

				vm.siteModifyShow = true;
				vm.siteModifyHidden = false;
				if(evt) evt.stopPropagation();
			},
			modifySiteInfoGPS() { 
				var vm = this,
					row = vm.currentSiteRow;

				Object.assign(vm.siteModifyForm,{
					serialNumber: row.serial_number,
					cellCode: row.small_cell_code,
					siteName: row.siteName,
					latitude: row.latitude,
					longitude: row.longitude
				});

				vm.siteModifyShow = true;
				vm.siteModifyHidden = false;
			},
			toInfoModifyGPS() {
				var vm = this,
					row = vm.detailForm;

				Object.assign(vm.siteModifyForm,{
					serialNumber: row.code,
					cellCode: row.cellCode,
					siteName: '',
					latitude: row.lat,
					longitude: row.lon
				});

				vm.siteModifyShow = true;
				vm.siteModifyHidden = false;
				vm.deviceDetailShow = false;
			},
			modifySiteSubmit() {
				var vm = this,
					url = '${ctx}/site/updateSiteInfo.action';

				vm.$refs.modifyForm.validate(function(r){
					if(r) {

						if(vm.siteModifyForm.serialNumber) {// save device latlon
							vm.setSiteDeviceLoc(vm.siteModifyForm);
						}else {// save site latlon
							axios.post(url, stringify(vm.siteModifyForm)).then((res)=>{

								vm.siteModifyShow = false;
								vm.$refs.siteList.refresh();
								vm.getSiteList();

								if(vm.siteModifyForm.siteName == vm.currentSiteRow.siteName) {
									Object.assign(vm.currentSiteRow, {
										longitude: vm.siteModifyForm.longitude,
										latitude: vm.siteModifyForm.latitude
									})
								}
							})
						}
					}
				})
			},
			delSite(row, evt) {
				var vm = this,
					url = '${ctx}/site/deleteSiteInfo.action',
					params = {
						siteName: row.siteName
					};

				evt.stopPropagation();

				vm.$confirm('<%=rb.getString("QueDingShanChuRenWu")%>', '<%=rb.getString("QueRen")%>',{
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
				}).then(() => {
					axios.post(url, stringify(params)).then((res)=>{
						
						vm.$refs.siteList.refresh();
						vm.getSiteList();
					})
				}).catch(()=>{})
			},
			modifyPickOnMAP() {
				var vm = this;

				vm.siteModifyHidden = true;
				// 检查是否使用 OpenLayers
				if (window.globSiteMap && typeof window.globSiteMap.on === 'function') {
					// OpenLayers 实现
					// 先移除旧的事件监听器
					if (vm.modifyClickHandler) {
						window.globSiteMap.un('singleclick', vm.modifyClickHandler);
					}
					// 创建新的事件处理器
					vm.modifyClickHandler = function(evt) {
						// 获取点击的坐标（经纬度）
						var coordinate = evt.coordinate; // [经度, 纬度] in map projection
						var lonLat = ol.proj.toLonLat(coordinate); // 转换为 [经度, 纬度]
						
						vm.siteModifyForm.latitude = lonLat[1].toFixed(6);
						vm.siteModifyForm.longitude = lonLat[0].toFixed(6);
						vm.siteModifyHidden = false;
						// 移除事件监听器
						window.globSiteMap.un('singleclick', vm.modifyClickHandler);
						vm.modifyClickHandler = null;
						// 恢复鼠标样式
						document.querySelector('#site_map').style.cursor = '';
					};
					// 绑定点击事件
					window.globSiteMap.on('singleclick', vm.modifyClickHandler);
					// 改变鼠标样式
					document.querySelector('#site_map').style.cursor = 'crosshair';
				}
			},
			addPickOnMap() {
				var vm = this;

				vm.siteAddHidden = true;
				
				// 检查是否使用 OpenLayers
				if (window.globSiteMap && typeof window.globSiteMap.on === 'function') {
					// OpenLayers 实现  先移除旧的事件监听器
					if (vm.addPickClickHandler) {
						window.globSiteMap.un('singleclick', vm.addPickClickHandler);
					}
					// 创建新的事件处理器
					vm.addPickClickHandler = function(evt) {
						// 获取点击的坐标（经纬度）
						var coordinate = evt.coordinate; // [经度, 纬度] in map projection
						var lonLat = ol.proj.toLonLat(coordinate); // 转换为 [经度, 纬度]
						
						vm.siteAddForm.latitude = lonLat[1].toFixed(6);
						vm.siteAddForm.longitude = lonLat[0].toFixed(6);
						vm.siteAddHidden = false;
						// 移除事件监听器
						window.globSiteMap.un('singleclick', vm.addPickClickHandler);
						vm.addPickClickHandler = null;
						// 恢复鼠标样式
						document.querySelector('#site_map').style.cursor = '';
					};
					// 绑定点击事件
					window.globSiteMap.on('singleclick', vm.addPickClickHandler);
					// 改变鼠标样式
					document.querySelector('#site_map').style.cursor = 'crosshair';
				} 
			},
			viewSiteDevice(row) {
				var vm = this,
					tRow = vm.thansformRow(row);

				Object.assign(vm.detailForm, tRow);
				vm.deviceDetailShow = true;
			},
			delSiteDevice(row) {
				var vm = this,
					params = {
						siteName: row.sub_station_name,
						serialNumber: row.serial_number
					},
					url = '${ctx}/site/deleteDeviceFromSite.action';

				var confirmStr = '<%=rb.getString("QueDingShanChuRenWu")%>'
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					axios.post(url, stringify(params)).then((res)=>{
						var data = res.data;

						if(data.success == true) {
							vm.$refs.viewTb.refresh();
						}
					})
				}).catch(() => {})
			},
			setSiteDeviceLoc(form) {
				var vm = this;

				$.ajax({
					url: '${ctx}/cell/topo/setLocationInfo.action',
					type: 'post',
					dataType: 'json',
					data: {
						longitude: form.longitude,
						latitude: form.latitude,
						setType: 'topo',
						cell_code: form.cellCode,
						operator_code: operator_code
					},
					success: function(data){
						if(data['success']){
							vm.siteModifyShow = false;
							vm.$refs.viewTb.refresh();

							if(form.siteName == vm.currentSiteRow.siteName) {
								Object.assign(vm.currentSiteRow, {
									longitude: vm.siteModifyForm.longitude,
									latitude: vm.siteModifyForm.latitude
								})
							}
						}
					}
				})
			},
			toAddSiteDevice() {
				var vm = this;

				vm.deviceToSiteForm.siteName = vm.currentSiteRow.siteName;
				vm.deviceToSiteType = vm.siteDefaultCk;
				vm.addSiteDeviceShow = true;
				vm.topolistShow = false;

				vm.$nextTick(function(){
					vm.$refs.deviceToSiteTb.clearSelection();
				})
			},
			addSiteDeviceSubmit() {
				var vm = this,
					url = '${ctx}/site/addDeviceToSite.action';

				vm.$refs.siteDeviceFm.validate(function(r){
					if(r) {
						axios.post(url, stringify(vm.deviceToSiteForm)).then((res)=>{

							vm.addSiteDeviceShow = false;
							vm.$refs.viewTb.refresh();
							vm.getSiteList();
						})
					}
				})
			},
			deviceToSiteTypeChange(val) {
				var vm = this;
				vm.$refs.deviceToSiteTb.clearSelection();
			},
			siteAddDeviceTypeChange(val) {
				var vm = this;
				vm.$refs.addTb.clearSelection();
			},
			toTOPOList(row) {
				var vm = this,
					rowData = row.row;

				vm.listOrTopoType = 'list';
				vm.topolistShow = true;

				if(['eNB'].includes(row.stationType)) {
					vm.euUrl = '${ctx}/cell/nxp/queryEUInfos.action';
					vm.ruUrl = '${ctx}/cell/nxp/queryRUInfos.action';
				}else if(row.stationType == 'gNB') {
					vm.euUrl = '${ctx}/gnb/gnbMonitor/getEUInfos.action';
					vm.ruUrl = '${ctx}/gnb/gnbMonitor/getRUInfos.action';
				}else if(row.stationType == 'GSM') {
					vm.btsUrl = '${ctx}/site/getBSCTupoInfos.action';
				}

				vm.queryParamsEURU.smallCellCode = row.small_cell_code;
				vm.queryParamsEURU.type = row.stationType;
			},

			tabClick(tab) {
				var vm = this;

				if(tab.name == 'site') {
					vm.siteListURL = '${ctx}/site/getSiteInfosList.action';
					vm.showSlider(null,false);
					/* 重置其他行箭头状态 */
					Array.from(document.querySelectorAll('.row-op')).map(function(item){
						item.classList.remove('el-icon-arrow-left');
						item.classList.add('el-icon-arrow-right');
					});
					vm.$nextTick(function(){
						vm.getSiteList();
					})
				}
			},
			refreshSiteMap() {
				var vm = this;

				vm.$refs.siteList.refresh();
			},
			getSiteList() {
				var vm = this,
					url = '${ctx}/site/getSiteInfosList.action',
					params = {};

				axios.get(url, stringify(params)).then(function(res){
					var nodeList = res.data;

					vm.allSiteLatlonNodes = vm.transformSiteNode(nodeList);
					vm.initSiteTopo();
				});
			},
			transformSiteNode(nodeList) {
				var vm = this;
				
				nodeList = nodeList.map(function(node){
					var lat = node.latitude,
						lon = node.longitude,
						contains = [];

					if(isNaN(node.latitude) || [null,undefined].includes(node.latitude)) {
						lat = '';
					}
					if(isNaN(node.longitude) || [null,undefined].includes(node.longitude)) {
						lon = '';
					}

					if(node.gsmCount && vm.siteDeviceType.includes('gsm')) contains.push('gsm');
					if(node.enbCount && vm.siteDeviceType.includes('enb')) contains.push('enb');
					if(node.gnbCount && vm.siteDeviceType.includes('gnb')) contains.push('gnb');

					return {
						code: node.siteName,
						olat: node.latitude,
						olon: node.longitude,
						lat: lat,
						lon: lon,
						type: 'site',
						contains: contains,
						gsmCount: node.gsmCount || 0,
						enbCount: node.enbCount || 0,
						gnbCount: node.gnbCount || 0,
						status: node.status,
						enbThroughput: node.enbThroughput || 0,
						gnbThroughput: node.gnbThroughput || 0,
						// test
						latitude: lat,
						longitude: lon,
						siteName: node.siteName,
					};
				});
				// 过滤无经纬度的节点
				nodeList = nodeList.filter(function(item){
					return item.lat && item.lon;
				});

				return nodeList;
			},
			initSiteTopo() {
				var vm = this,
					nodes = vm.allSiteLatlonNodes || [];

				// 清除旧地图容器
				var mapContainer = document.getElementById('site_map');
				if(mapContainer) {
					mapContainer.innerHTML = '';
				}
				
				// 清除旧地图对象（兼容Leaflet和OpenLayers）
				if(globSiteMap) {
					// 清理测距资源
					if (window.globOLHelper && typeof window.globOLHelper.cleanupMeasureResources === 'function') {
						window.globOLHelper.cleanupMeasureResources();
					}
					
					if(typeof globSiteMap.setTarget === 'function') {
						// OpenLayers 地图
						globSiteMap.setTarget(null);
					} else if(typeof globSiteMap.remove === 'function') {
						// Leaflet 地图
						globSiteMap.remove();
					}
					globSiteMap = null;
				}

				// 处理基站数据格式
				var resObj = vm.proccessNodes(nodes);
				
				// 设置地图图层源
				var tileUrl = offlineMapEnable 
					? '${ctx}/map/{z}/{x}/{y}.png'
					: 'https://{a-c}.tile.openstreetmap.org/{z}/{x}/{y}.png';
				
				// 创建瓦片图层
				var tileLayer = new ol.layer.Tile({
					source: new ol.source.XYZ({
						url: tileUrl,
						maxZoom: offlineMapEnable ? 12 : 18
					}),
					visible: true
				});
				
				// 创建地图实例
				var siteMap = new ol.Map({
					target: 'site_map',
					layers: [tileLayer],
					view: new ol.View({
						center: ol.proj.fromLonLat([resObj.lon, resObj.lat]),
						zoom: 10,
						maxZoom: offlineMapEnable ? 12 : 18,
						minZoom: 4
					})
				});
				globSiteMap = siteMap;
				// 绘制 Site 图层
				vm.createSiteLayer(resObj.enb, siteMap);
				
				// 自适应视图到节点边界
				if(resObj.enb.length > 0) {
					var extent = ol.extent.boundingExtent(
						resObj.enb.map(function(node) {
							return ol.proj.fromLonLat([parseFloat(node.lon), parseFloat(node.lat)]);
						})
					);
					siteMap.getView().fit(extent, {
						padding: [50, 50, 50, 50],
						maxZoom: offlineMapEnable ? 12 : 16
					});
				}
				// 事件绑定
				siteMap.on('moveend', function(ev) { // 移动结束后重新计算
					vm.siteFilterRender(resObj.enb, siteMap, true);
				});
			},
			rowdblclickSite(row, evt) {
				var code = row.siteName,
					valid = this.allSiteLatlonNodes.map(function(row){
						return row.code;
					}).includes(code);

				// topo中有该节点时，高亮居中显示
				if(valid && row.latitude && row.longitude){
					if(typeof globSiteMap.getView === 'function') {
						// OpenLayers 地图
						var view = globSiteMap.getView();
						view.animate({
							center: ol.proj.fromLonLat([row.longitude-0, row.latitude-0]),
							zoom: 15,
							duration: 500
						});
					} else {
						// Leaflet 地图
						globSiteMap.setView([row.latitude-0, row.longitude-0], 3);
					}
					setTimeout(function(){
						highlightSiteNode(code)
					},500)
				}
			},
			getSiteLimitedNodes(nodes, bufferEnable) {
				return nodes;
			},
			siteFilterRender(nodes, siteMap, bufferEnable) {
				var vm = this,
					threshold = this.limit;

				// 检查地图类型
				if(typeof siteMap.getView === 'function') {
					// OpenLayers 地图
					var view = siteMap.getView(),
						extent = view.calculateExtent(siteMap.getSize()),
						sw = ol.proj.toLonLat([extent[0], extent[1]]),
						ne = ol.proj.toLonLat([extent[2], extent[3]]);

					// 删除历史节点图层和overlays
					var layersToRemove = [];
					siteMap.getLayers().forEach(function(layer){
						if(layer.get('layerType') === 'site') {
							layersToRemove.push(layer);
						}
					});
					layersToRemove.forEach(function(layer){
						siteMap.removeLayer(layer);
					});
					
					var overlaysToRemove = [];
					siteMap.getOverlays().forEach(function(overlay){
						if(overlay.get('overlayType') === 'site' || overlay.get('overlayType') === 'site-popup') {
							overlaysToRemove.push(overlay);
						}
					});
					overlaysToRemove.forEach(function(overlay){
						siteMap.removeOverlay(overlay);
					});

					if(nodes.length > threshold) {
						var inviewNodes = nodes.filter(function(item){
							var lat = parseFloat(item.lat),
								lon = parseFloat(item.lon);
							return sw[1] <= lat && lat <= ne[1] && sw[0] <= lon && lon <= ne[0];
						});
						vm.createSiteLayer(inviewNodes, siteMap, bufferEnable);
					}else {
						vm.createSiteLayer(nodes, siteMap, bufferEnable);
					}
				}
			},
			createSiteLayer(nodes, siteMap, bufferEnable) {
				var vm = this;
				// 判断可视区域内的节点是否超过阀值
				nodes = this.getSiteLimitedNodes(nodes, bufferEnable);
				
				// 检查地图类型
				if(typeof siteMap.getView === 'function') {
					// OpenLayers 地图 - 使用Overlay渲染节点（参考getSiteOptions的setIcon方式）
					
					// 先移除所有旧的Site节点Overlay，防止重复
					var overlaysToRemove = [];
					siteMap.getOverlays().forEach(function(overlay) {
						if(overlay.get('overlayType') === 'site-node') {
							overlaysToRemove.push(overlay);
						}
					});
					overlaysToRemove.forEach(function(overlay) {
						siteMap.removeOverlay(overlay);
					});
					
					var colorMap = {
						'1': 'normal',
						'2': 'abnormal',
						'3': 'offline'
					};
					
					nodes.forEach(function(node) {
						// 获取状态类名
						var statusClass = colorMap[node.status] || 'normal';
						
						// 根据状态确定颜色类
						var colorClass = statusClass === 'normal' ? 'blue-bg' : 
										 statusClass === 'abnormal' ? 'abnormal-color' : 
										 'offline-color';
						
						// 创建节点HTML（参考getSiteOptions的setIcon），状态类添加到node-item-leaf上
						var html = ['<div class="node-item-leaf ' + statusClass + '">',
										'<span class="node-code">' + node.code + '</span>',
										'<i class="el-icon el-icon_tpopo_site ' + colorClass + '" style="font-size: 30px;"></i>',
										'<div class="site-device-tag">',
											node.contains.includes('gsm') && vm.isGSMEnable?'<i class="el-icon el-icon_tpopo_2G ' + colorClass + '"></i>':'',
											node.contains.includes('enb')?'<i class="el-icon el-icon-topo-enb ' + colorClass + '"></i>':'',
											node.contains.includes('gnb')?'<i class="el-icon el-icon_tpopo_5G ' + colorClass + '"></i>':'',
										'</div>',
									'</div>'].join('');
						
						var nodeEl = document.createElement('div');
						nodeEl.className = 'airport-icon site-node-marker';
						nodeEl.innerHTML = html;
						nodeEl.style.cursor = 'pointer';
						
						// 创建Overlay
						var overlay = new ol.Overlay({
							element: nodeEl,
							positioning: 'center-center',
							stopEvent: false,
							offset: [0, 0]
						});
						
						overlay.setPosition(ol.proj.fromLonLat([parseFloat(node.lon), parseFloat(node.lat)]));
						overlay.set('nodeData', node);
						overlay.set('overlayType', 'site-node');
						siteMap.addOverlay(overlay);
						
						// 点击事件（参考onEachRecord的click逻辑）
						nodeEl.addEventListener('click', function(evt) {
							// 如果使用OpenLayers测距，不需要手动处理，OpenLayers会自动处理
							if (window.globOLHelper && window.globOLHelper.measureActive) {
								return;
							}
							highlightSiteNode(node.code);
							vm.siteType = 'view';
							vm.viewDeviceURL = '${ctx}/cell/topo/getDeviceInfoListForSite.action';
							vm.resetSiteStatus();
							vm.siteInfoShow = true;
							vm.siteAddHidden = false;
							vm.currentSiteRow = {
								siteName: node.code,
								latitude: node.lat,
								longitude: node.lon,
								status: node.status,
								enbCount: node.enbCount || 0,
								gnbCount: node.gnbCount || 0,
								gsmCount: node.gsmCount || 0,
								enbThroughput: node.enbThroughput || 0,
								gnbThroughput: node.gnbThroughput || 0
							};
							vm.viewQuery.siteName = node.code;
							evt.stopPropagation();
						});
						
						// 悬浮事件（参考onEachRecord的mouseover逻辑）
						var popupOverlay = null;
						nodeEl.addEventListener('mouseenter', function(evt) {
							// 创建popup
							var popupEl = document.createElement('div');
							popupEl.className = 'ol-popup site-popup';
							popupEl.innerHTML = vm.popoverSiteInfo(node);
							popupEl.style.zIndex = '999999'; // 确保浮层在所有节点之上
							
							popupOverlay = new ol.Overlay({
								element: popupEl,
								positioning: 'bottom-center',
								stopEvent: false,
								offset: [0, -25], // 再下移10px (从-35改为-25)
								className: 'site-popup-overlay'
							});
							
							popupOverlay.set('overlayType', 'site-popup');
							popupOverlay.setPosition(ol.proj.fromLonLat([parseFloat(node.lon), parseFloat(node.lat)]));
							siteMap.addOverlay(popupOverlay);
							
							// 手动设置overlay容器的z-index
							setTimeout(function() {
								var overlayContainer = popupEl.parentElement;
								if(overlayContainer) {
									overlayContainer.style.zIndex = '999999';
								}
							}, 0);
						});
						
						nodeEl.addEventListener('mouseleave', function(evt) {
							if(popupOverlay) {
								siteMap.removeOverlay(popupOverlay);
								popupOverlay = null;
							}
						});
					});
					$(window).resize();
				}
			},
			popoverSiteInfo(node) {
				var vm = this,
					content = [];
				
				// 构建HTML内容 - 添加site-popup-content类名用于特定样式
				content.push('<div class="popup-content site-popup-content">');
				content.push('<a href="#" class="ol-popup-closer" onclick="event.preventDefault(); this.parentElement.parentElement.style.display=\'none\';">×</a>');
				// Site名称显示在title中，带省略号
				content.push('<h3 class="site-popup-title" title="' + node.code + '">' + node.code + '</h3>');
				// 纬度
				content.push('<div class="info-row">');
				content.push('<span class="info-label"><%=rb.getString("WeiDu")%>:</span>');
				content.push('<span class="info-value">' + node.lat + '</span>');
				content.push('</div>');
				// 经度
				content.push('<div class="info-row">');
				content.push('<span class="info-label"><%=rb.getString("JingDu")%>:</span>');
				content.push('<span class="info-value">' + node.lon + '</span>');
				content.push('</div>');
				// eNB Count
				if(vm.enbEnable) {
					content.push('<div class="info-row">');
					content.push('<span class="info-label">eNB Count:</span>');
					content.push('<span class="info-value">' + (node.enbCount || 0) + '</span>');
					content.push('</div>');
				}
				// gNB Count
				if(vm.gnbEnable) {
					content.push('<div class="info-row">');
					content.push('<span class="info-label">gNB Count:</span>');
					content.push('<span class="info-value">' + (node.gnbCount || 0) + '</span>');
					content.push('</div>');
				}
				// GSM Count (如果启用)
				if(vm.isGSMEnable) {
					content.push('<div class="info-row">');
					content.push('<span class="info-label">GSM Count:</span>');
					content.push('<span class="info-value">' + (node.gsmCount || 0) + '</span>');
					content.push('</div>');
				}
				// eNB Throughput
				if(vm.enbEnable) {
					content.push('<div class="info-row">');
					content.push('<span class="info-label">eNB Throughput(Mbps):</span>');
					content.push('<span class="info-value">' + (node.enbThroughput || 0) + '</span>');
					content.push('</div>');
				}
				// gNB Throughput
				if(vm.gnbEnable) {
					content.push('<div class="info-row">');
					content.push('<span class="info-label">gNB Throughput(Mbps):</span>');
					content.push('<span class="info-value">' + (node.gnbThroughput || 0) + '</span>');
					content.push('</div>');
				}
				content.push('</div>');

				return content.join('');
			},
			changeListOrTopo(type) {
				var vm = this;

				vm.listOrTopoType = type;
				if(type == 'topo') {
					vm.$nextTick(function(){
                        var params = { 
							domId: 'mapSiteTopoPage',
                            smallCellCode: vm.queryParamsEURU.smallCellCode,
                            ShanChu: '<%=rb.getString("ShanChu")%>',
                            QueRen: '<%=rb.getString("QueRen")%>',
                            QueDingShanChuSheBei: '<%=rb.getString("QueDingShanChuSheBei")%>',
                            ChengGong: '<%=rb.getString("ChengGong")%>',
                            KPISheBei: '<%=rb.getString("KPISheBei")%>'
						};
						initMapSiteTopo(params);
					});
				}
			},
			siteCheckAllChange(val) {
				var vm = this
					allCodes = [];

				if(vm.isGSMEnable) {
					allCodes.push('gsm');
				}
				if(vm.enbEnable) {
					allCodes.push('enb');
				}
				if(vm.gnbEnable) {
					allCodes.push('gnb');
				}

				vm.isIndeterminate = false;
				vm.siteDeviceType = val? allCodes : [];

				vm.getSiteList();
			},
			siteCheckChange() {
				var vm = this,
					length = vm.siteDeviceType.length,
					mathLength = 0;

				if(vm.isGSMEnable) {
					mathLength += 1;
				}
				if(vm.enbEnable) {
					mathLength += 1;
				}
				if(vm.gnbEnable) {
					mathLength += 1;
				}

				vm.siteCheckAll = length == mathLength;
				vm.isIndeterminate = length < mathLength && length > 0;

				vm.getSiteList();
			},

			thansformRow(node) {
				var vm = this;
				var enable = node.mme_enable == 1? true:false,
					mmePool_1 = node.mme_pool_1?node.mme_pool_1:'',
					mmePool_2 = node.mme_pool_2?node.mme_pool_2:'',
					serverList = node.s1siglinkserverlist?node.s1siglinkserverlist:'',
					lat = node.latitude,
					lon = node.longitude;
				if(isNaN(node.latitude) || [null,undefined].includes(node.latitude)) {
					lat = '';
				}
				if(isNaN(node.longitude) || [null,undefined].includes(node.longitude)) {
					lon = '';
				}
				// 规范属性
				return {
					code: node.serial_number,
					olat: node.latitude,
					olon: node.longitude,
					lat: lat,
					lon: lon,
					name: node.host_name,
					groupName: node.group_name,
					ip: node.cell_ip,
					type: 'enb',
					online: node.connection_status? 'on':'off',
					active: node.op_state? 'yes':'no',
					alarmCount: node.alarm_count,
					alarmLevel: node.alarm_serverity,
					mmeEnable: enable,
					mmeStatus: node.mme_status,
					mmePool1: mmePool_1,
					mmePool2: mmePool_2,
					serverList: serverList,
					cellCode: node.small_cell_code,
					alarm: node.alarm,
					sasEnable: node.sas_enable,
					sasState: node.sas_state,
					relaCpeList: node.relaCpeList,
					NEW_MME_STATUS: node.new_mme_status,
					ueCount: node.ue_count,
					isGSM: node.isGSM
				};
			},
			toUETrace(sn) {
				var vm = this;
					
				vm.uetraceURL = '${ctx}/cell/topo/toUETrace.action';
				vm.ueSlideTitle = 'SN：' + sn;
				vm.$refs.uetrace.showSlide(function(){
					eventBus.$emit("eu-trace", sn);
				});
			},
			closeUETrace() {
				var vm = this;
				vm.$refs.uetrace.hide();
			},
            showTopoSettingPanel(type) {
                var vm = this,
                    str = Math.random().toString(),
                    rowData = {},
                    params = { 
                        timeZone: timeZone,
						isDual: false,
						page: 1,
						rows: 50
                    };
                
                // 根据类型设置查询参数
                if(type === 'enb' || type === 'gnb' || type === 'gsm'){
                    params.small_cell_code = vm.infoForm.cellCode;
                }else if(type === 'cpe'){
                    params.cpe_code = vm.infoForm.cellCode;
                }
                if(type === 'gsm'){
                    params.product_model = 'BSC,BTS'
                }
                if(type === 'gnb'){
                    params.isGnb = 1;
                    params.monitor = 1;
                }
                
                // 根据类型选择不同的 API 接口
                var apiUrl = '';
                if(type === 'cpe'){
                    apiUrl = '${ctx}/cell/CPE/queryCpeInfosList.action?type=0';
                }else{
                    // enb, gnb, gsm 使用同一个接口
                    apiUrl = '${ctx}/cell/cpeinfos/queryCpeInfosList.action';
                }
                
                // 发起请求获取设备数据
                axios.post(apiUrl, stringify(params)).then(function(res){
                    // 检查接口返回数据是否有效
                    if(res.data === null || (typeof res.data === 'object' && Object.keys(res.data).length === 0)) {
                        vm.$message.error('<%=rb.getString("SheBeiBuCunZaiHuoYiBeiShanChu")%>');
                        return;
                    }
                    rowData = res.data;
                    // 根据类型设置不同的 URL 和事件
                    var settingUrl = '';
                    var eventName = '';
                    var eventParams = [];
                    
                    if(type === 'enb'){
                        // ENB 类型
                        enbPlatform = rowData.platform_flag;
                        enbPlatformType = rowData.platformType;
                        settingUrl = '${ctx}/enb/setting/openSettingPage.action?randomValue=' + str;
                        eventName = 'row-data';
                        eventParams = [rowData, 'info', 'topo'];
                        
                        // 检查 enodeb_monitor.jsp 是否已打开 setting.jsp 页面
                        if(typeof enbvm !== 'undefined') {
                            try {
                                enbvm.$refs.settingPage.hide();
                            } catch(e) {}
                        }
                    }else if(type === 'cpe'){
                        // CPE 类型 - 设置 sessionStorage（参照 cpeTopo_tab.jsp 的 showCpeTopoSettingPanel 方法）
                        sessionStorage.setItem('oldProduct', rowData.OLDPRODUCT || '');
                        sessionStorage.setItem('CONNECTION_STATUS', rowData.CONNECTION_STATUS || '');
                        sessionStorage.setItem('CPE_CODE', rowData.CPE_CODE || '');
                        sessionStorage.setItem('PRODUCT', rowData.PRODUCT || '');
                        sessionStorage.setItem('cpeName', rowData.CPE_NAME || '');
                        sessionStorage.setItem('lanFlag', rowData.LAN_INTERFACE || '');
                        sessionStorage.setItem('cpeSN', rowData.SERIAL_NUMBER || '');
                        sessionStorage.setItem('softVersion', rowData.SOFTWARE_VERSION || '');
                        sessionStorage.setItem('imsi', rowData.IMSI || '');
                        sessionStorage.setItem('model_name', rowData.MODEL_NAME || '');
                        sessionStorage.setItem('module_name', rowData.MODEL_NAME || '');
                        
                        settingUrl = '${ctx}/cpe/setting/openSettingPage.action?randomValue=' + str;
                        eventName = 'row-data';
                        eventParams = [rowData.CPE_CODE,rowData.MACADDRESS,rowData.SERIAL_NUMBER,rowData.connection_status,rowData,'topo'];
                        
                        // 检查 cpe_monitor_vue.jsp 是否已打开 setting.jsp 页面
                        if(typeof cpevm !== 'undefined') {
                            try {
                                cpevm.$refs.cpeSettingPage.hide();
                            } catch(e) {}
                        }
                    }else if(type === 'gnb'){
                        // GNB 类型
                        settingUrl = '${ctx}/gnb/setting/openSettingPage.action?randomValue=' + str;
                        eventName = 'action-settingPage';
                        eventParams = [rowData, 'info', 'topo'];
                        
                        // 检查 gnodeb_monitor.jsp 是否已打开 setting.jsp 页面
                        if(typeof gnbMonitor !== 'undefined') {
                            try {
                                gnbMonitor.$refs.slide.hide();
                            } catch(e) {}
                        }
                    }else if(type === 'gsm'){
                        // GSM 类型
                        settingUrl = '${ctx}/cell/cpeinfos/toGSMMonitorSettingPages.action?randomValue=' + str;
                        eventName = 'gsm-setting-row';
                        eventParams = [rowData, 'info', 'eNBTopo_tab'];
                        
                        // 检查 gsm_monitor_vue.jsp 是否已打开 gsm_setting.jsp 页面
                        if(typeof gsmvm !== 'undefined') {
                            try {
                                gsmvm.$refs.settingPage.hide();
                            } catch(e) {}
                        }
                    }
                    // 打开设置页面滑动面板
                    vm.slideURL = settingUrl;
                    vm.slidePosition = 'top';
                    vm.sliderHeight = '100%';
                    vm.sliderWidth = '72%';
                    vm.$refs.topoSettingSlide.showSlide(function(){
                        eventBus.$emit(eventName, ...eventParams);
                    });
                }).catch(function(error){});
            },
            hideTopoSettingSlide() {
                var vm = this;
                hideAllComboBoxPanels();
                vm.$refs.topoSettingSlide.hide();
            },
			initLevel() {
				var vm = this;
				axios.get('${ctx}/cell/topo/getAllKpiLevels.action').then(function(res){
					var data = res.data.data;
					if(data) {
						vm.kpiLevels5 = (data.kpiLevels5 || []).map(function(item){
							return {
								level: item.level,
								color: item.color,
								label: item.label,
								range: [item.range_min, item.range_max]
							};
						});
						vm.kpiLevelsDL = (data.kpiLevelsDL || []).map(function(item){
							return {
								level: item.level,
								color: item.color,
								label: item.label,
								range: [item.range_min, item.range_max]
							};
						});
						vm.kpiLevelsUL = (data.kpiLevelsUL || []).map(function(item){
							return {
								level: item.level,
								color: item.color,
								label: item.label,
								range: [item.range_min, item.range_max]
							};
						});
					}
				});
			}
		},
		mounted() {
			var vm = this;
			vm.initLevel();
			// 确保页面加载时退出KPI模式（防止页面刷新后仍保持KPI渲染）
			vm.kpiMode = false;
			vm.kpiPanelShow = false;
			// 清除可能残留的KPI显示元素
			if (typeof $ !== 'undefined') {
				$('.node-kpi-value').remove();
			}
			// 通知 OLHelper 清除KPI模式
			if (window.globOLHelper && window.globOLHelper.clearKpiMode) {
				window.globOLHelper.clearKpiMode();
			}
			
			// 载入电子围栏数据（只加载到内存，不显示在地图上）
			// 实际的显示会在地图初始化完成后自动进行（见 loadTopoData 方法）
			if(vm.isFenceEnable) {
				vm.getFenceEnableStatus();
			}

			// 初始化全局UE状态设置
			window.topoUeStatusSettings = vm.statusForm.ueStatus || [];
			
			// 初始化全局Name状态设置
			window.topoNameStatusSettings = vm.statusForm.nameStatus || [];
			
			// 初始化全局statusForm配置，用于判断节点状态显示优先级
			window.topoStatusForm = vm.statusForm;
			
			// 根据支持状态动态初始化设备类型列表
			var deviceTypes = []; // eNB 默认启用
			if (vm.enbEnable) {
				deviceTypes.push('enb');
			}
			if (vm.gnbEnable) {
				deviceTypes.push('gnb');
			}
			if (vm.isGSMEnable) {
				deviceTypes.push('gsm');
			}
			if (vm.cpeEnable) {
				deviceTypes.push('cpe');
			}
			vm.statusForm.deviceType = deviceTypes;
			
			// 获取sas开关状态
			axios.post('${ctx}/cell/topo/getSasEnableStatus.action',stringify({operator_code: operator_code})).then(function(res){
				if(res && res.data) {
					vm.sasEnable = !res.data.success || writableMap.CODE_ENB_MONITOR !== true;
				}
			});

			vm.initOverView();

			if(window.topoOverViewTimer) {
				clearInterval(window.topoOverViewTimer);
			}
			window.topoOverViewTimer = setInterval(vm.initOverView, 600000);

			$('.flex-bt-cls').on('click', function(evt) {
				var target = $(evt.target);
				$('.flex-bt-cls').removeClass('selected');
				target.addClass('selected');
			});
			
			// 监听设置页面的关闭事件
			eventBus.$off('cancel-enb-topo-setting').$on('cancel-enb-topo-setting', function(){
				vm.hideTopoSettingSlide();
			});
			
			// 监听全屏状态变化（包括通过鼠标移到顶部点击退出按钮）
			var fullscreenChangeHandler = function() {
				var isFullscreen = document.fullscreenElement || 
				                   document.mozFullScreenElement || 
				                   document.webkitFullscreenElement || 
				                   document.msFullscreenElement;
				
				if (!isFullscreen) {
					// 关闭所有展开的下拉框
					hideAllComboBoxPanels();
					
					// 退出全屏时，将所有弹窗和下拉框移回 body
					vm.restoreDialogsToBody();
					
					// 清理观察器
					if(vm._dialogObserver) {
						vm._dialogObserver.disconnect();
						vm._dialogObserver = null;
					}
				}
			};
			
			// 添加多浏览器兼容的全屏变化监听器
			document.addEventListener('fullscreenchange', fullscreenChangeHandler);
			document.addEventListener('webkitfullscreenchange', fullscreenChangeHandler);
			document.addEventListener('mozfullscreenchange', fullscreenChangeHandler);
			document.addEventListener('MSFullscreenChange', fullscreenChangeHandler);
		}
	});

	/** 鼠标移入，显示当前MME详细信息   * @param ele{dom}：容器dom节点 * @param index{number}：mme状态值 **/
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
		
		var YiLianJie = '<%= rb.getString("MMEYiLianJie")%>';
		var WeiLianJie = '<%= rb.getString("MMEWeiLianJie")%>';
		var eleClass = $(ele).text(); 
		$(".mmeDetails").hide();
		
		if(eleClass == 'MME1'){
			$(ele).siblings(".MME1Detail").fadeToggle();
			if(index == 1){
				$(".mme1Stauts").text(YiLianJie);
			}else if(index == 0){
				$(".mme1Stauts").text(WeiLianJie);
			}
		}else if(eleClass == 'MME2'){
			$(ele).siblings(".MME2Detail").fadeToggle();
			if(index == 1){
				$(".mme2Stauts").text(YiLianJie);
			}else if(index == 0){
				$(".mme2Stauts").text(WeiLianJie);
			}
		}else if(eleClass == 'MME'){
			$(ele).siblings(".MMEDetail").fadeToggle();
			if(index == 1){
				$(".mmeStauts").text(YiLianJie);
			}else if(index == 0){
				$(".mmeStauts").text(WeiLianJie);
			}
		}	
	}
</script>

