<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<link rel="stylesheet" href="${ctx}/js/topo/leaflet.css" type="text/css" media="screen" />
<link rel="stylesheet" href="${ctx}/js/topo/dvf.css" type="text/css" media="screen"/>
<link rel="stylesheet" href="${ctx}/js/topo/node.css" type="text/css" media="screen"/>
<style>		
	.statisticDiv > p input{
		width:310px !important;
	}
	.echart_enb{
		width:90%;
		height:326px;
		margin-top:30px;
		padding-bottom:20px; 
	}
	.kpiNameStyle{
		font-size:14px;
		color:#333333;
		margin-top:30px;
		font-weight: bold;
	}
	.echartStyle{
		width:90%;
		height:310px;
		margin-top:10px;
		border:1px solid #e9e9e9;
	}
	.alarmList{
		width:235px;
		height:260px;
		background:#FFFFFF;
		position:absolute;
		left:1px;
		z-index:9;
		display:none;
		padding:30px 0 0 30px;
		-webkit-box-shadow: 2px 4px 15px 0 rgba(70, 128, 172, 0.35);
		-moz-box-shadow: 2px 4px 15px 0 rgba(70, 128, 172, 0.35);
		box-shadow: 2px 4px 15px 0 rgba(70, 128, 172, 0.35);
	}
	.alarmList label{
		margin-left:25px;
	}
	.alarmList .alarmSelectAll{
		height:45px;
	}
	.alarmList .alarmSingleCheckBox li{
		height:35px
	}
	.flex-ctn-info {
		display: flex;
		flex-direction: column;
	}
	.flex-item-info {
		flex: 1 auto;
		overflow: auto;
	}
	.statisticDiv.loading::before {
		background-color: #fff;
	}
	.el-card__header{
		background:#fff;
		box-shadow:none;
		border-bottom:1px solid #EEE;
		height:48px;
		line-height:48px;
		color:#363B4E;
		font-size:16px;
		padding:0 20px;
		font-weight:bold;
	}
	.slidebarSecondTabsContainer li{
		height:36px;
		line-height:36px;
		width:120px;
		margin-right:0;
		text-align:left;
		color:#666;
		border-bottom:1px solid #EEE;
		float:none;
		display:block;
		font-size:12px;
	}
	.slidebarSecondTabsContainer li.active{
		background:#EDF6FF;
		color:#4D84FF;
		border-bottom:1px solid #EEE;
		box-sizing:unset;
	}
	.windowButtonGroup .linkbutton > span{
		min-width:auto;
	}
	.form-group{
		margin-bottom:15px;
	}
	.el-icon-operation-download-failed:before{
		color:#E88282;
	}
	.blockTabsTitleTabsContainer {
		border:none;
		border-bottom:1px solid #E9E9E9;
		margin-bottom:20px;
	}
	.blockTabsTitleTabsContainer li {
		font-size:14px;
		color:#363B4E;
		padding:0;
		margin-right:40px;
	}
	.blockTabsTitleTabsContainer li.active {
		color:#4D84FF;
		background:unset;
		border-bottom:1px solid #4D84FF;
	}
</style>
<style>
	.configuration-area {
		display: flex;
		height:100%;
		overflow: auto;;
		border-left:1px solid #E9E9E9;
		flex: 1 auto;
	}
	.split-left-area,.split-right-area,.split-line-area {
		margin-top: 10px;
		display: flex;
		flex-direction: column;
		flex: 1 auto;
		width: 46%;
	}
	.split-line-area {
		padding: 0 10px 10px 10px;
		width: 80%;
		margin-right: 40px;
	}
	.split-left-area > div {
		flex: 1 auto;
		margin: 10px;
		border: 1px solid #E9E9E9;
		border-top: 2px solid #4D84FF;
		border-radius: 5px;
		padding: 10px;
	}
	.split-right-area {
		margin-right: 40px;
		padding: 10px;
	}
	.split-right-area > div,.split-line-area > div {
		flex: 1 auto;
		border: 1px solid #E9E9E9;
		border-top: 2px solid #4D84FF;
		padding: 10px;
	}
	.split-right-area > div,.split-line-area > div {
		border-radius: 5px;
		margin-top: 20px;
	}
	.split-right-area > div:first-child {
		margin-top: 0px;
	}
	.area-title {
		padding-left: 5px;
		font-size: 14px;
		font-weight: bold;
	}
	.area-list {
		padding-top: 10px;
		display: flex;
		flex-wrap: wrap;
	}
	.area-item-cls {
		position: relative;
		margin: 5px 5px 10px 15px;
		flex: 1 auto;
		width: 45%;
		display: flex;
		word-break: break-all;
	}
	.area-item-cls.vertic {
		width: 80%;
	}
	.area-item-cls::before {
		content: attr(label);
		min-width: 140px;
		color: #999999;
	}

	.kpiPeriod {
		display:inline-block;
		vertical-align:middle;
		margin-left:20px;
	}
	.kpiPeriod span{
		display:inline-block;
		border:1px solid #e9e9e9;
		height:30px;
		line-height:30px;
		text-align:center;
		width:60px;
		cursor:pointer;
		margin-left:-5px;
	}
	.kpiPeriod span:first-of-type{
		border-radius: 4px 0 0 4px;
		margin-left:-4px;
	}
	.kpiPeriod span:last-of-type{
		border-radius:0 4px 4px 0;
		margin-left:-4px;
	}
	.kpiPeriod span.active {
		color:#1913bb;
		border-color:#1913bb;
		position:relative;
	}
	.type-change-cls {
		position: absolute;
		left: 30px;
	}
	.type-change-cls span {
		height: 23px;
		line-height: 23px;
	}
	.type-change-cls span:first-child {
		border-radius: 15px 0 0 15px;
	}
	.type-change-cls span:last-child {
		border-radius: 0 15px 15px 0;
	}
	.el-icon-topo-enb.white-bg {
		border: none !important;
	}
	.el-icon-topo-enb.white-bg::after{
		content: '';
		padding: 8px;
		background: #fff; 
		position: absolute;
		top: 4px;
		left: 0px;
		border-radius: 8px;
		z-index: -1;
	}
	.split-right-area .leaflet-top {
		position: relative !important;
	}
	.label-title {
		position: relative;
	}
	.label-title::before {
		content: attr(label);
		position: absolute;
		top: -20px; 
		font-weight: bold;
	}
	.group-bt {
		position: absolute;
		display: flex;
		top: 5px;
		right: 5px;
	}
	.group-bt > a {
		cursor: pointer;
		padding: 2px 15px;
		border: 1px solid #e9e9e9;
		border-left-width: 0px;
	}
	.group-bt > a.active {
		color: #1913bb;
		border: 1px solid #69b6fc;
		font-weight: bold;
	}
	.group-bt > a:hover {
		color: #69b6fc;
	}
	.group-bt > a:first-child {
		border-left-width: 1px;
		border-radius: 15px 0 0 15px;
	}
	.group-bt > a:last-child {
		border-radius: 0 15px 15px 0;
	}
	.enbDeviceReportLogsDiv .el-icon-status-terminate:before,
	.enbDeviceReportLogsDiv .el-icon-status-waiting1:before,
	.enbDeviceReportLogsDiv .el-icon-status-inProgress:before{
		color:#4D84FF;
	}
	.enbDeviceReportLogsDiv .el-icon-status-success:before {
		color:#67D972;
	}
	.enbDeviceReportLogsDiv .el-icon-status-failed:before {
		color:#E88282;
	}
</style>
<div id="gnodeb_detail_ctn" style='width:100%;height:100%;display:flex;flex-direction:column;flex:1 1 auto'>
	<div style='width:100%;height:100%;background:#fff;display:flex;'>
		<div class="slidebarSecondTabsTitle" style='width:auto;'>
			<ul class="slidebarSecondTabsContainer titleTabsList" style='height:100%;'>
				<li tabtit="configurationDiv" onclick="turnTabs(this);" class="active" logtype="con"><%=rb.getString("HalobCanShuPeiZhi")%></li>
				<li tabtit="statisticDiv" onclick="turnTabs(this);" logtype="op"><%=rb.getString("LiShi")%></li>
				<li tabtit="gnbLogsDiv" onclick="turnTabs(this);" logtype="sys" class=""><%=rb.getString("RiZhi")%></li>
				<li tabtit="gnbLicenseDiv" onclick="turnTabs(this);viewLicense();" logtype="license" class="">License</li>
				<li tabtit="flowDiv" onclick="turnTabs(this);" logtype="flow"><%=rb.getString("LiuLiangTongJi")%></li>
			</ul>
		</div>
		<div style='height:100%;overflow:hidden;border-left:1px solid #E9E9E9;flex:1;'>
			<!-- Configuration -->
			<div id="configuration_div" class="configurationDiv" style="width: 100%;height: 100%;display: flex;overflow: auto;flex-wrap: wrap;">
				<div class="split-left-area">
					<div>
						<div class="area-title"><%=rb.getString("SASSheBeiXinXi")%></div>
						<div class="area-list">
							<div class="area-item-cls" label="<%=rb.getString("XiaoZhanBianMa")%>">${serialNumber}</div>
							<div class="area-item-cls" label="<%=rb.getString("SoftwareVersion")%>">${softwareVersion}</div>
							<div class="area-item-cls" label="<%=rb.getString("ChanPinLeiXingBiaoZhi")%>">${productType}</div>
							<div class="area-item-cls" label="<%=rb.getString("FirmwareVersion")%>">${hardwareVersion}</div>
							<div class="area-item-cls" label="<%=rb.getString("SheBeiZu")%>">${groupName}</div>
							<div class="area-item-cls" label="<%=rb.getString("YunXingShiJian")%>">${systemUpTime}</div>
							<div class="area-item-cls" v-show="false" label="Model Name" v-html="modelType"></div>
							<div class="area-item-cls" label="<%=rb.getString("DiYiCiLianJieShiJian")%>">${firstPeriodTime}</div>
							<div class="area-item-cls" label="<%=rb.getString("ShangCiLianJieShiJian")%>">${lastPeriodTime}</div>
						</div>
					</div>
					<div>
						<div class="area-title"><%=rb.getString("ZhuangTai")%></div>
						<div class="area-list">
							<div class="area-item-cls vertic" label="<%=rb.getString("ShiFouJiHuo")%>" v-html="activeStatus"></div>
						</div>
					</div>
					<div>
						<div class="area-title"><%=rb.getString("WangLuoSheZhi")%></div>
						<div class="area-list">
							<div class="area-item-cls vertic" label="<%=rb.getString("IPDiZhi")%>">${ipAddress}</div>
						</div>
					</div>
				</div>
				<div class="split-right-area">
					<div style="border-top: 2px solid #4D84FF;max-height: 120px;">
						<div class="area-title"><%=rb.getString("XiaoQuXinXi")%></div>
						<div class="area-list">
							<div class="area-item-cls" label="<%=rb.getString("HostName")%>">${cellName}</div>
							<div class="area-item-cls" label="TAC">${tac}</div>
							<div v-if="false" class="area-item-cls" label="NRCGI">${eci}</div>
							<div class="area-item-cls" label="PCI">${pci}</div>
						</div>
					</div>
					<div v-if="false" style="max-height: 180px;min-height: 180px;position: relative;">
						<div class="area-title"><%=rb.getString("WeiZhi")%></div>
						<div class="area-list">
							<div class="area-item-cls vertic" label="<%=rb.getString("JingDu")%>">${gpsLongitude}</div>
							<div class="area-item-cls vertic" label="<%=rb.getString("WeiDu")%>">${gpsLatitude}</div>
							<div class="area-item-cls vertic" label="<%=rb.getString("GaoDu")%>">${gpsHeight}</div>
						</div>
						<div id="gps_map" style="position: absolute; border: 1px solid silver;right: 15px; top: 15px; bottom: 15px; left: 240px;">
						
						</div>
					</div>
				</div>
			</div>
			<div class="splitPage statisticDiv defaultInformationInput">
				<div class="splitPageContent">
					<div class="splitGroup">
						<div class="splitGroup_title" onclick="resizeStatisChart();"><%=rb.getString("ZhuangTaiXinXi")%></div>
						<div class="splitGroup_body">
							<!-- 下拉框信息 -->
							<div class="selectInfo" style="display: none;">
								<select id="selectType" class="easyui-combobox border border-box" data-options="editable:false" style="height:30px;width:190px;"></select>
							</div>
							<div label="gNB Online Status" class="label-title" style="display: flex;width: 90%;height: 346;border: 1px solid #e9e9e9;">
								<div class="echart_enb" id="echart_gnb_online" style="border: none;margin-top: 0px;"></div>
								<div class="echart_enb" id="echart_gnb_online_time" style="margin-top: 0px;max-width: 33%;border: none;"></div>
							</div>
							<div label="gNB Cell Status" class="label-title" style="display: flex;width: 90%;height: 346;border: 1px solid #e9e9e9;margin-top: 30px;">
								<div class="echart_enb" id="echart_gnb_active" style="border: none;margin-top: 0px;"></div>
								<div class="echart_enb" id="echart_gnb_active_time" style="margin-top: 0px;max-width: 33%;border: none;"></div>
							</div>
						</div>
					</div>
				</div>
			</div>
			
			<!-- 日志信息界面 -->
			<div class="gnbLogsDiv" style='width:100%;height:100%;display:flex;flex-direction:column'>
				<div class="splitPageContent" style="display:flex;flex-direction:column;padding:20px 20px 0 20px;">
					<ul class="blockTabsTitleTabsContainer titleTabsList">
						<li tabtit="enbDeviceReportLogsDiv" onclick="turnTabs(this)" class="active" logtype="op"><%=rb.getString("SheBeiRiZhi")%></li>
					</ul>
					<div class="enbDeviceReportLogsDiv" style="flex:1;display:block;">
						<el-ctable
							ref="ctableLog"
							:url="logUrl"
							:query-params="queryParams"
							row-key="serial_number">
							<el-table-column prop="op" label=" " width="50" align="center">
								<template slot-scope="scope">
								<div class="el-icon el-icon-operation-more" v-clickoutside="hideMenus" @click="opClick(scope.row,event)"></div>
								</template>
							</el-table-column>
							<el-table-column prop="serial_number" label="<%=rb.getString("JiZhanXuLieHao")%>"></el-table-column>
							<el-table-column prop="task_status" label="<%=rb.getString("ShouJiZhuangTai")%>" width="180">
								<template slot-scope="scope">
								<div v-if="scope.row.execute_type == 'Immediately'">						
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
								</div>
								</template>
							</el-table-column>
							<el-table-column label='<%=rb.getString("ShiBaiYuanYin")%>' prop="failureReason" show-overflow-tooltip="true"></el-table-column>
							<el-table-column prop="update_time" label="<%=rb.getString("LogsGengXinShiJian")%>"></el-table-column>>
						</el-ctable>
						<!-- 菜单 -->
						<el-cmenu ref="menu"
						@click="menuClick"
						:data="menus"></el-cmenu>
					</div>
				</div>
			</div>
			<!-- license信息界面 -->
			<div id='licenseInfo' class='gnbLicenseDiv' style='width:100%;height:100%;display:flex;flex-direction:column'>
				<div class="splitPageTitle">
					<span><%=rb.getString("LicenseXinXi")%></span>
				</div>
				<div class="splitPageContent" style="display:flex;flex-direction:column;padding:20px;">
					<div class='license_basic_info' >
						<div class="form-group">
							<div class="group-title" onclick="titleExted(this)">
								<span class="title-icon"></span>
								<span class="title-text"><%=rb.getString("JiBenXinXi")%></span>
							</div>
							<div class="form-wrap">
								<div class="form-item">
									<label  class="inputTittleCss"><%=rb.getString("JiZhanXuLieHao")%><%=rb.getString("MaoHao")%></label>
									<input id='serialNumberInput'  class="inputDivCss border border-box" disabled='true'/>
								</div>
								<div class="form-item">					
									<label  class="inputTittleCss"><%=rb.getString("LicenseBanBen")%><%=rb.getString("MaoHao")%></label>
									<input id='versionInput'  class="inputDivCss border border-box" disabled='true'/>
								</div> 
								<%-- <div class="form-item">					
									<label  class="inputTittleCss"><%=rb.getString("ZuoZhe")%><%=rb.getString("MaoHao")%></label>
									<input id='authorInput'  class="inputDivCss border border-box" disabled='true'/>
								</div> --%>
								<div class="form-item">					
									<label class="inputTittleCss"><%=rb.getString("ShengChengShiJian")%><%=rb.getString("MaoHao")%></label>
									<input id='timeInput' class="inputDivCss border border-box"  disabled="true"/>
								</div>  
								<%-- <div class="form-item">					
									<label  class="inputTittleCss"><%=rb.getString("LicenseMoShi")%><%=rb.getString("MaoHao")%></label>
									<input id='modeInput'  class="inputDivCss border border-box" style="height:26px;" disabled="true"/>
								</div> --%>	
							</div>
						</div>
					</div>
					<div class='license_capacity_info' >
						<div class="form-group last">
							<div class="group-title" onclick="titleExted(this)">
								<span class="title-icon"></span>
								<span class="title-text"><%=rb.getString("NengLiLieBiao")%></span>
							</div>
							<div class='license_capacity_content' style='height:360px;margin-top:20px;'>
								<table id='license_table' ></table>
							</div>
						</div>
					</div>
				</div>
			</div>
			<!-- 统计信息界面 -->
			<div class="flowDiv">
					<el-ctable ref="flowTb"
						:url="flowUrl"
						:query-params="flowQueryParams"
						row-key="serialNumber" height="100%" width="100%">
						<div slot="toolbar" style="padding: 0 15px;">
							<el-select v-model="flowType" @change="flowTypeChange">
								<el-option v-for="item in flowOpts" :label="item.label" :value="item.value"></el-option>
							</el-select>
						</div>
						<el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' min-width="130" prop="serialNumber"></el-table-column>
						<el-table-column label='<%=rb.getString("KaiShiShiJian")%>' min-width="100" prop="startTime" sortable></el-table-column>
						<el-table-column label='<%=rb.getString("JieShuShiJian")%>' min-width="100" prop="endTime"></el-table-column>
						<el-table-column label='UEID' min-width="100"  prop="ueId"></el-table-column>
						<el-table-column label='PLMNID' v-if="flowType=='plmn'" prop="breakoutValue"></el-table-column>
						<el-table-column label='Nssai' v-if="flowType=='nssai'" prop="breakoutValue"></el-table-column>
						<el-table-column label='Target IP' v-if="flowType=='fiveTuple'" prop="breakoutValue"></el-table-column>
						<el-table-column label='Domain' v-if="flowType=='domain'" prop="breakoutValue"></el-table-column>
						<el-table-column label='<%=rb.getString("ZongLiuLiang")%>' prop="totalFlow"></el-table-column>
					</el-ctable>
			</div>
		</div>
	</div>
</div>

<script type="text/javascript">
	var timeParam = getNowTimeToZoneTimeRange(timeZone, 144),
		start_time_gnb = timeParam.start_time.substring(0,11)+"00:00:00",
		end_time_gnb = timeParam.end_time,
		smallCellCode = '${smallCellCode}',
		pointerCount = 6*24+1;// 24*(60/5);

	new Vue({
		el: '#gnodeb_detail_ctn',
		data() {

			return {
				modelType: '',
				logUrl: '/cell/collect/getImmediateCollectLogTaskPageList.action',
				queryParams: {
					isGnb: 1,
					search_text: '',
					device_type: 'eNB',
					device_code: '${smallCellCode}',
					like_fields: 'serial_number',
					timeZone: timeZone,
					start_time: '',
					end_time: ''
				},
          		menus: [],
				flowType: 'plmn',
				flowOpts: [
					{label: '<%=rb.getString("PLMNFenLiu")%>', value: 'plmn'},
					{label: '<%=rb.getString("WanLuoQiePianFenLiu")%>', value: 'nssai'},
					{label: '<%=rb.getString("IPFenLiu")%>', value: 'fiveTuple'},
					{label: '<%=rb.getString("YuMingFenLiu")%>', value: 'domain'}
				],
				flowUrl: '${ctx}/gnb/gnbMonitor/getlocalBreakoutInfosList.action',
				flowQueryParams: {
					timeZone: timeZone,
					serialNumber: '${serialNumber}',
					breakoutType: 'plmn'
				}
			}
		},
		computed: {
			activeStatus() {
				return cellStateFormatter('${activeStatus}');
			}
		},
		methods: {
			init() {
				loadAjax('echart_gnb_online','offline',function(){
					setTimeout(function(){
						var dom = document.querySelector('#echart_gnb_online');
						if(dom) {
							var chart = echarts.getInstanceByDom(dom);
							// 初始化时间轴切换事件
							chart.off('timelinechanged');
							chart.on('timelinechanged',function(p){
								initPieOnline(getYesterDay(6-p.currentIndex),'day');
							});
						}
					},0)
				});

				loadAjax('echart_gnb_active','active',function(){
					setTimeout(function(){
						var dom = document.querySelector('#echart_gnb_active');
						if(dom) {
							var chart = echarts.getInstanceByDom(dom);
							// 初始化时间轴切换事件
							chart.off('timelinechanged');
							chart.on('timelinechanged',function(p){
								initPieActive(getYesterDay(6-p.currentIndex),'day');
							});
						}
					},0)
				});
				
				initPieActive(getYesterDay(0),'day');
				initPieOnline(getYesterDay(0),'day');
			},
			opClick(row,ev){//软件升级点击操作出现下拉菜单  1.等待   2.进行中  3.暂停  4.已结束 5.终止中 6.暂停中
				var vm = this;

				<%-- vm.taskStatus = row.task_status;
				vm.menus= [
						{label:'<%=rb.getString("ZhongZhiRenWu")%>',cls:"el-icon el-icon-operation-terminate" ,code:'terminate', row: row},
						{label:'<%=rb.getString("XiaZai")%>',cls:"el-icon el-icon-operation-download" ,code:'download', row: row},
						{label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete",code:'del', row: row}
					];

				initTaskStatus(status,vm.menus); --%>
				
				var status = row.task_status;  
		          //0-收集未开始；1-正在收集；2-收集完成；3-收集失败；4-收集终止;5-等待上传； 
		          //表格处理 0-等待， 1-进行中，2-成功，3-失败，4-终止，5-正在停止上报（进行中），6- 停止上报 成功（终止），7-停止上报失败（进行中），8-重启(等待)
		          vm.taskStatus = row.task_status;
		    
		          var terminateFlag = false , showFlag = true , delFlag = false;
		    
		          if(status == 0 || status == 1 || status == 5 || status == 7 || status == 8){
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

		          vm.menus= [
		                {label:'<%=rb.getString("ZhongZhiRenWu")%>',cls:"el-icon el-icon-operation-terminate CODE_ENB_LOGS hidden" ,code:'terminate',disable:!terminateFlag ,show:showFlag, row: row},
		                {label:'<%=rb.getString("XiaZai")%>',cls:"el-icon el-icon-operation-download" ,code:'download', row: row},
		                {label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete CODE_ENB_LOGS hidden",code:'del',disable:!delFlag, row: row}
		          ];

				vm.$nextTick(function(){
					document.body.click();
					vm.showMenus(ev);
				});
			},
			/**
			*  获取点击项数据
			* @parame ev:点击属性数据
			*/
			menuClick(item){ //单点击方法 -- 设备上报日志、告警日志 
				var vm = this,
					codes = {
						terminate: vm.terminateCollectTask,
						download: vm.downlodFile,
						del: vm.delCollectFile
					},
					key = item.code,
					row = item.row;

				if(codes[key]) codes[key](row);
			},
			terminateCollectTask(row) {
				var vm = this , 
					url = '${ctx}/cell/collect/goTerminateImmediateCollectLogFile.action',
					status = row.task_status,
					params = {
						taskIds: row.task_id,
						device_code: row.device_code,
						execute_type: row.execute_type,
						isGnb: 1
					};

				if (status != 0 && status != 5 && status != 1 && status != 7 && status != 8) {
					showMsg('prompt_msg','<%=rb.getString("MeiYouKeTingZhiShouJiDeSheBei")%>');
					return;
				}
				
				axios.post(url, stringify(params)).then(function(response){
					var data = response.data;
					if(data["success"]){
					vm.$refs.ctableLog.refresh()
					vm.$message({
						type:'success',
						message:'<%=rb.getString("ChengGong")%>'
					})
					}else{
					vm.$message.error(data["message"])
					}
				}).catch(function(error){});
			},
			/**
			*  下载文件
			* @param row:当前数据
			*/
			downlodFile(row){ 
				var vm = this,
					taskId = row.task_id,
					fileNum = row.file_num,
					fileName = row.file_name||'',
					params = {
						taskIds: taskId,
						timeZone: timeZone,
						fileName: fileName,
						isGnb: 1
					},
					checkURL = '${ctx}/cell/collect/getDownloadFileNumber.action',
					url = "${ctx}/cell/collect/doDownloadImmediateCollectLogFile.action";
				
				axios.post(checkURL,stringify(params)).then(function(response){
					var data = response.data;
					if(data.length > 0 || data.fileNum > 0){
						vm.createForm(url, params);
					}else{
						vm.$message.error('<%=rb.getString("WenJianBuCunZai")%>')
					}
				}).catch(function(error){})
			},
			/**
			* 删除任务  -- 设备上报日志 、告警日志 
			* @param row:当前数据
			*/
			delCollectFile(row){
				var vm = this , 
					url='${ctx}/cell/collect/doClearImmediateCollectLogFile.action' ,
					id = row.task_id,
					fileName = ''
					params= {
						taskIds:id,
						fileName : fileName,
						isGnb: 1
					};

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
						vm.$refs.ctableLog.refresh()
						vm.$message({
							type:'success',
							message:'<%=rb.getString("ChengGong")%>'
						})
						}else{
						vm.$message.error(data["message"])
						}
					}).catch(function(error){
						
					})
				}).catch()
			},
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
			},
			hideMenus() {
				this.$refs.menu.hide()
			},
			showMenus(evt) {
				this.$refs.menu.show(evt)
			},
			flowTypeChange(val) {
				this.flowQueryParams.breakoutType = val;
			}
		},
		mounted() {
			this.init();
		}
	});

	/**
	* 请求后台统计图表数据
	* @param idName{number}：图表容器Id
	* @param type{number}：类型
	* @param fn{number}：回调方法
	**/
	function loadAjax(idName,type,fn){
		dataArr=[];
		var params = {
				isGnb: 1,
				device_type: "enb",
				device_code: smallCellCode,
				time_level: "min",
				timeZone: timeZone,
				start_time: start_time_gnb,
				end_time: end_time_gnb
	 	}
		// 获取统计数据
		$.post("${ctx}/system/device/getDeviceStatusDataList.action", params, function(data){
			if(data){
			}else{
				data = [];
			}
			dataArr=[];
			$.each(data,function(index,item){
				var xDate=item.statistics_time.split(' ');
				var activeSta=item.active_status;
				var onlineSta=item.online_status;
				var ueSta=item.ue_count;
				if(activeSta == 0){
					activeSta = activeSta+1;
				}else if(activeSta == 1){
					activeSta = activeSta+2
				}
				if(onlineSta == 0){
					onlineSta = onlineSta+1;
				}else if(onlineSta == 1){
					onlineSta = onlineSta+2
				}
				dataArr[index]=[xDate[0],xDate[1],activeSta,onlineSta,ueSta];
			});
			// 初始化系列数据
			finalArr = [];
			for(var initIndex=0;initIndex<7;initIndex++ ){
				var arrIndex = getYesterDay(initIndex);
				if(!finalArr[arrIndex]){
					var timeaxisArr = [], activeaxisArr = [],offlineaxisArr=[],ueaxisArr=[];
					for(var axisIndex=0; axisIndex<pointerCount; axisIndex++){
						var yaxisStartTime = getYesterDay(initIndex).substring(0,10)+' 00:00:00',
							offsetTime = addTimes(new Date(yaxisStartTime),axisIndex*10);
						timeaxisArr.push(formatDate(offsetTime));
						if(offsetTime.getTime() > new Date(end_time_gnb).getTime()){
							activeaxisArr.push(undefined);
							offlineaxisArr.push(undefined);
							ueaxisArr.push(undefined);
						}else{
							activeaxisArr.push(1);
							offlineaxisArr.push(1);
							ueaxisArr.push(0);
						}
					}
					finalArr[arrIndex] = [];
					finalArr[arrIndex]['time'] = timeaxisArr;
					finalArr[arrIndex]['active'] = activeaxisArr;
					finalArr[arrIndex]['offline'] = offlineaxisArr;
					finalArr[arrIndex]['ue'] = ueaxisArr;
				}
			}
			// 初始化系列的x、y轴数据
			$.each(dataArr,function(n,m){
				var dateIndex = m[0],timeValue=m[1].substring(0,5),activeValue=m[2],offlineValue=m[3],ueValue=m[4];
				if(finalArr[dateIndex]['time'].includes(dateIndex+' '+m[1])){
					var differTimes = new Date(dateIndex+' '+m[1]).getTime() - new Date(dateIndex+' 00:00:00').getTime();
					var axisDateIndex = Math.round(differTimes/(1000*60*10));
					finalArr[dateIndex]['active'].splice(axisDateIndex,1,activeValue);
					finalArr[dateIndex]['offline'].splice(axisDateIndex,1,offlineValue); 
					finalArr[dateIndex]['ue'].splice(axisDateIndex,1,ueValue); 
				}
			});

			var preData = null;
			for(var key in finalArr){
				if(typeof finalArr[key] != 'function'){
					if(preData){
						finalArr[key]['active'][pointerCount-1] = preData['active'][0];
						finalArr[key]['offline'][pointerCount-1] = preData['offline'][0];
						finalArr[key]['ue'][pointerCount-1] = preData['ue'][0];
						preData = finalArr[key]; 
					}else preData = finalArr[key];
				}
			}
			if(fn) try{fn();}catch(e){}
			createEchart(finalArr,idName,type);
		}, "json");
	}
	/**
	* 绘制统计页面图表
	* @param finalArr{array}：初始化后的图表系列数据
	* @param idName{string}：图表容器Id
	* @param type{string}：类型
	**/
	function createEchart(finalArr,idName,type){
		var preValue;
		$("#"+idName).css("display","block");
		//$("#"+idName).siblings(".echart_enb").css("display","none");
		var enodebChart = echarts.init(document.getElementById(idName));
	 	 var option = {
				baseOption:{
					 timeline: {
			              axisType: 'category',  
			              show: true,  
			              autoPlay: false,  
			              playInterval: 1000,  
			              data: [getYesterDay(6).substring(5).replace("-","."),
			                     getYesterDay(5).substring(5).replace("-","."),
			                     getYesterDay(4).substring(5).replace("-","."),
			                     getYesterDay(3).substring(5).replace("-","."),
			                     getYesterDay(2).substring(5).replace("-","."),
			                     getYesterDay(1).substring(5).replace("-","."),
			                     getYesterDay(0).substring(5).replace("-",".")],
			              notMerge:true,
			              controlPosition:'none',
			              currentIndex:6,
			              checkpointStyle:{
			            	  color:'#209FFF',
			            	  borderColor:'none'
			              },
						  lineStyle : { color : '#B0AFBA',width : 1 },
						  itemStyle : {
						  	  normal : { borderColor : '#B0AFBA' },
							  emphasis : {
								  borderColor : '#1e90ff',
								  color : '#1e90ff'
							  }
						  },
			              label:{
			            	  emphasis:{
			            		  color:'#209FFF'
			            	  }
			              }
			          },
			           grid:{
			        	  top:50,
			        	  bottom:80
			          },
					  color: ['#69b6fc','#DCDFE6'],
			          tooltip:{
			        	  trigger:"axis",
			              padding:[10,5,10,5],
			              formatter:function(params){
			            	  if(params.length){
			            		  var ueStr = "";
				            	  var str = "";
				            	   if(idName == "echart_enb_ue"){
				            		  ueStr += "<div>" + params[0].name + "</div>";
				            		  if(params[0].data == undefined){
				            			  ueStr += "<div>UE:-</div>" 
				            		  }else{
				            			  ueStr += "<div>UE:" + params[0].data + "</div>"
				            		  }
				            		  return ueStr;
				            	  }else{
				            		  str += "<div>" + params[0].name + "</div>";
				            		  return str;
				            	  }
			            	  }
			              }
			          },
			          xAxis: [{ 
		            	   name:"<%=rb.getString("XiaoShi") %>",
				    	   type:"category",
				    	   axisLabel:{
					    		  show:true,
					    		  formatter : function(val) {
					                	var secondTime = val.split(' ')[1];
					            		clock = secondTime.substring(0,2);
					                 if(secondTime.substring(3,5)=='00') return clock;
					                 return val;
					              },
		                          interval:function(index){
		                        	  if(index%6 == 0 && index !=pointerCount-1){
		                        		  return true;
		                        	  }
		                          }
					    	 },
							axisLine:{
								lineStyle:{ color:"#666666" }
							},
							splitLine:false ,
							boundaryGap:false
		              }],
		              yAxis: { 
		            	    minInterval: (idName=='echart_enb_ue')?1:1,
		            	    max: (idName=='echart_enb_ue')?null:4,
		            	    name: (idName=='echart_enb_ue')?"<%=rb.getString("GeShuUe") %>":"<%=rb.getString("ZhuangTai") %>", 
							type:"value",
							axisLine:{
								show:true,
								lineStyle:{ color:"#666666" },
								interval:12
							},
							axisTick:{
								show:false
							},
							splitLine:{
								show:idName =='echart_enb_ue'?true:false
							},
							axisLabel:{
								show:true,
								formatter:function(value,index){
									if(idName == 'echart_gnb_active'){
										var texts=[];
										if(value == 3){
											texts.push("<%=rb.getString("ShouYe_HuoYue")%>");
										}else if(value == 1){
											texts.push("<%=rb.getString("ShouYe_BuHuoYue")%>");
										}
										return texts;
									}else if(idName == 'echart_gnb_online'){
										var texts=[];
										if(value == 3){
											texts.push("<%=rb.getString("ShouYe_ZaiXian")%>");
										}else if(value == 1){
											texts.push("<%=rb.getString("ShouYe_BuZaiXian")%>");
										}
										return texts;
									}else if(idName == 'echart_enb_ue'){
										return value;
									}
								}
							}
		              },
		              series: [  
				                  {  
				                		type:'line',
				                		step:idName=='echart_enb_ue'?false:true,
				    		        	 markLine: idName=='echart_enb_ue'?{}:{
					                	  	data:[{
					                	  		name:' ',
					                	  		yAxis: 1
					                	  	},
					                	  	{
					                	  		name:' ',
					                	  		yAxis: 3
					                	  	}],
					                	  	animation: false,
					                	  	lineStyle:{
					                	  		normal:{
					                	  			type:'dashed',
					                	  			color:'#ccc',
					                	  			width:0.5,
					                	  			opacity:0.8
					                	  		}
					                	  	},
					                	  	symbolSize:0,
					                	  	silent:true,
					                	  	label:{
					                	  		normal:{
					                	  			show:false
					                	  		}
					                	  	}
					                	},
					                  symbolSize: 0
				                  }
				              ]  
				},	
				//变量则写在options中  
				options:[  
					{  
						xAxis: [{ 
							data:finalArr[getYesterDay(6)]['time']
						}],
						series: [
							{  
							data:finalArr[getYesterDay(6)][type]
							} 
						]  
					},  
					{  
						xAxis: [{ 
							data:finalArr[getYesterDay(5)]['time']
						}],
						series: [ 
							{  
							data:finalArr[getYesterDay(5)][type]
							} 
						]  
					},
					{  
						xAxis: [{  
							data:finalArr[getYesterDay(4)]['time']
						}], 
						series: [  
							{  
							data:finalArr[getYesterDay(4)][type]
							} 
						]  
					},
					{  
						xAxis: [{  
							data:finalArr[getYesterDay(3)]['time'],
							
						}],
						series: [
							{    
							data:finalArr[getYesterDay(3)][type]
							} 
						]  
					},
					{  
						
						xAxis: [{  
							data:finalArr[getYesterDay(2)]['time']
						}],
						series: [  
							{  
							data:finalArr[getYesterDay(2)][type]
							} 
						]  
					},
					{  
						xAxis: [{  
							data:finalArr[getYesterDay(1)]['time']
						}], 
						series: [ 
							{   
							data:finalArr[getYesterDay(1)][type]
							}  
						]  
					},
					{  
						
						xAxis: [{  
							data: finalArr[getYesterDay(0)]['time']
						}],
						series: [
							{    
							data:finalArr[getYesterDay(0)][type]
							} 
						]  
					}
				]
			  } 
		 enodebChart.setOption(option);
	 	 enodebChart.resize();
	}
	// 初始化柱状图
	function initPieActive(time,type) {
		var params = {
			isGnb: 1,
			timeZone: timeZone,
			time: time,
			timeType: type,
			smallCellCode: smallCellCode
		};
		$.post('${ctx}/system/device/getEnbStatisticsCountDataList.action',params,function(res){
			var curDay = gloableTime.substring(0,10),
				hourse = gloableTime.substring(11,13),
				hourseTotal = curDay==time?hourse:24,
				minites = curDay==time?(Math.ceil(gloableTime.substring(14,16)/10)*10):0,
				activeTotal = res[0]?(res[0].active_minute||0):0,
				activeRate = (activeTotal*100/(hourseTotal*60+minites)).toFixed(0),
				data = [{name:'<%=rb.getString("JiHuoShiChange")%>',value: activeRate},{name:'<%=rb.getString("WeiJiHuoShiChange")%>',value: 100-activeRate}],
				activechart = echarts.init(document.querySelector('#echart_gnb_active_time'));
			
			var options = {
				color: ['#69b6fc','#f5f5f5'],
				tooltip: {
					trigger: 'item',
					formatter: '{b}: {c}%'
				},
				series: [
					{
						name: '',
						type: 'pie',
						radius: '55%',
						center: ['50%','50%'],
						data: data,
						emphasis: {
							itemStyle: {
								shadowBlur: 10,
								shadowOffsetX: 0,
								shadowColor: 'rgba(0,0,0,0.5)'
							}
						},
						label: {
							normal: {
								color: '#333',
								fontSize: 10
							}
						}
					}
				]
			};

			activechart.setOption(options);
		},'json');
	}

	function initPieOnline(time,type) {
		var params = {
			isGnb: 1,
			timeZone: timeZone,
			time: time,
			timeType: type,
			smallCellCode: smallCellCode
		};
		$.post('${ctx}/system/device/getEnbStatisticsCountDataList.action',params,function(res){
			var curDay = gloableTime.substring(0,10),
				hourse = gloableTime.substring(11,13),
				hourseTotal = curDay==time?hourse:24,
				minites = curDay==time?(Math.ceil(gloableTime.substring(14,16)/10)*10):0,
				onlineTotal = res[0]?(res[0].online_minute||0):0,
				onlineRate = (onlineTotal*100/(hourseTotal*60+minites)).toFixed(0),
				data = [{name:'<%=rb.getString("ZaiXianShiJian")%>',value: onlineRate},{name:'<%=rb.getString("DuanKaiShiJian")%>',value: 100-onlineRate}],
				onlinechart = echarts.init(document.querySelector('#echart_gnb_online_time'));
			
			var options = {
				color: ['#69b6fc','#f5f5f5'],
				tooltip: {
					trigger: 'item',
					formatter: '{b}: {c}%'
				},
				series: [
					{
						name: '',
						type: 'pie',
						radius: '55%',
						center: ['50%','50%'],
						data: data,
						emphasis: {
							itemStyle: {
								shadowBlur: 10,
								shadowOffsetX: 0,
								shadowColor: 'rgba(0,0,0,0.5)'
							}
						},
						label: {
							normal: {
								color: '#333',
								fontSize: 10
							}
						}
					}
				]
			};

			onlinechart.setOption(options);
		},'json');
	}
	/**
	* 判断对象是否为空
	* @param obj{object}: 要判断的对象
	**/
	function isEmptyObject(obj){
		for(var key in obj){
			return false;
		}
		return true;
	}
	/**
	* 查看基站license信息
	* @param divId{string}: license查看容器Id
	**/
	function viewLicense(divId){
		var small_cell_code = smallCellCode;
		var param ={
				isGnb: 1,
				small_cell_code: small_cell_code
			}
		// 获取license信息
		$.post("${ctx}/cell/license/getHalobLicenseInfo.action", param, function(data) {
			if(!isEmptyObject(data)){
				$('#serialNumberInput').val(data.serial_number);
				$('#versionInput').val(data.version);
				$('#timeInput').val(data.generate_date);
				$('#modeInput').val(data.halob_mode);

				var licenseData= [];
				if(data.capacity_list){
					$.each(data.capacity_list,function(index,item){
						var obj={};
						obj.Capacity = item.id;
						obj.ValidPeriod = item.valid_period;
						var remperiod = item.remaining_period;
						if (item.valid_period =="0"){
							obj.RemainingPeriod = "<%=rb.getString("Yongjiu")%>";
						}
						else{
							obj.RemainingPeriod = remperiod;
						}
						obj.quantity = item.capa_value;
						obj.Description = item.description;
						licenseData.push(obj);
					})
				}

				$("#license_table").datagrid({
						border:false,
						fit: true,
						fitColumns: true,
						singleSelect:true,
						striped:true,
						rownumbers:false,
						pagination:false,
						columns:[[
							{ field:'Capacity',title:'<%=rb.getString("TeXingID")%>',width:50},   
							{ field:'Description',title:'<%=rb.getString("MiaoShu")%>',width:120},
							{ field:'quantity',title:'<%=rb.getString("ShuLiang")%>',width:50},   
							{ field:'ValidPeriod',title:'<%=rb.getString("YouXiaoQi")%>',width:60},
							{ field:'RemainingPeriod',title:'<%=rb.getString("ShengYuShiJian")%>',width:60 },
										
						]],
						data:licenseData
					});
			}else{
				$("#license_table").datagrid({
						border:false,
						fit: true,
						fitColumns: true,
						singleSelect:true,
						striped:true,
						rownumbers:false,
						pagination:false,
						columns:[[
							{ field:'Capacity',title:'<%=rb.getString("TeXingID")%>',width:50},  
							{ field:'Description',title:'<%=rb.getString("MiaoShu")%>',width:50},
							{ field:'ValidPeriod',title:'<%=rb.getString("YouXiaoQi")%>',width:100},
										
						]],
						data:[]
					});
			}
		}, "json");
	}
</script>
