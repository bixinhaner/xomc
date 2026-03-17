<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<script type="text/javascript" src="${ctx}/js/element/Sortable.min.js?_=${omc_ver}"></script>
<script type="text/javascript" src="${ctx}/js/element/vuedraggable.umd.min.js?_=${omc_ver}"></script>
<style>

    #falutListPage span[class*='status_']{
        padding-left:25px;
    }
	.upsMonitorSlideCls .title-text:after{
		display: none
	}
	.upsMonitorSlideCls .group-title{
		margin:10px
	}
	.upsMonitorSlideCls .el-card__body{
		display:flex;
		flex-direction:column;
		flex:1 1 auto;
		overflow:auto;
		padding: 0;
	}
	.upsMonitorSlideCls .footer{
		width: 100%;
    	overflow: overlay;
    	border-top: #EEEEEE solid 1px;
		position: absolute;
		bottom: 0px;
		height: 48px;
		line-height: 48px;
		
	}
	.upsMonitorSlideCls .footer .linkbutton{
		margin-left: 48px
	}
	.ml20{
		margin-left:20px
	}
	.plr15{
		padding: 0 15px !important
	}
	.plr15 .el-form-item__label{
		width:100px;
		float:left;
	}
	.el-form-item{
		margin-bottom:22px
	}
	#falutListPage{
		height:100%;
		background:#FFFFFF
	}
	.h100{
		height:100%
	}
	.ml10{
		margin-left:10px
	}
	.curpo{
		cursor:pointer
	}
	.w290{
		width:290px
	}
	#falutListPage .wendu-middle .el-icon-status-temperature:before ,#falutListPage .el-icon-status-run:before{
		color:#F2B354
	}
	#falutListPage .sfp-wait .el-icon-status-SFP:before ,#falutListPage .lan-wait .el-icon-status-LAN:before{
		color:#CFCFCF	
	}	
	#falutListPage .el-icon-status-powerON:before ,#falutListPage .sfp-success .el-icon-status-SFP:before ,#falutListPage .wendu-low .el-icon-status-temperature:before ,#falutListPage .el-icon-status-high:before ,#falutListPage .lan-success .el-icon-status-LAN:before{
		color:#67D972
	}
	#falutListPage .el-icon-status-powerOFF:before ,#falutListPage .sfp-error .el-icon-status-SFP:before ,#falutListPage .wendu-height .el-icon-status-temperature:before ,#falutListPage .el-icon-status-low:before ,#falutListPage .lan-error .el-icon-status-LAN:before{
		color:#E88282
	}
	.f12{
		font-size:12px
	}
	#falutListPage .txc{
		text-align: center
	}
	#falutListPage .alarmListSty{
		display: inline-block;
		width: 25px;
		height:25px;
		line-height: 25px;
		border-radius: 50%;
		color: #FFFFFF;
		font-size:12px
	}
	#falutListPage .alarmCritical{
    	background:#E88282;
	}
	#falutListPage .alarmMajor{
		background:#DCAA5E;
	}
	#falutListPage .alarmMinor{
		background:#CCCC66;
	}
	#falutListPage .alarmWarning{
		background:#9AF0FE;
	}
	#falutListPage .mr5{
		margin-right:5px
	}
	.upsMonitorSlideCls .linkbuttonGroup{
		margin-left:50px
	}
	.upsMonitorSlideCls .el-form-item:after{
		display:none !important
	}
	.upsMonitorSlideCls .form-group .el-form-item__label{
		line-height: 28px
	}
	#falutListPage .texc{
		text-align: center
	}
	#falutListPage .showHideItem {
		position:absolute;
		background:white;
		z-index:888;
		padding-top: 10px;
		padding-left: 10px;
		width: 600px;
		top: 65px;
		left: 0px;
		display: none;
		box-shadow:5px 10px 23px 0px rgba(0,0,0,0.15);
	}
	#falutListPage .showHideItem input{
		margin-top:-2px;
		margin-bottom:1px;
		vertical-align:middle;
		margin-right:20px;
	}
	#falutListPage .selectAll{
		height:32px;
		width:334px;
		padding:28px 0px 0px 30px;
	}
	#falutListPage .showHideItem .select-all-cls {
		padding: 10px 0 0 9px;
		display: flex;
		align-items: center;
	}
	#falutListPage .showHideItem .select-all-cls > i {
		margin-right: 5px;
	}
	#falutListPage .showHideItem .select-all-cls > span {
		font-size: 14px;
		font-weight: bold;
		margin-left: 10px;
	}
	#falutListPage .showHideItem .el-icon-close1::before {
		color: #333;
	}

	#falutListPage .list-group > span {
		display: flex;
		flex-direction: column;
		flex-wrap: wrap;
		padding-left: 15px; 
		height: 230px;
	}
	#falutListPage .list-group-item {
		display: inline-block;
		position: relative;
		padding: 0px 10px;
		margin: 3px 5px;
		width: 215px;
		border: 0px dashed #ddd;
		cursor: move;
	}
</style>

<div class="overflow-cls">
<div id="falutListPage" class="pageDefault commonWarp" style="min-width: 900px;">
	<!-- 按钮  -- 导出 -->
	<div slot="reference" class="circleIcon placeholder-bt" placeholder="<%=rb.getString("DaoChu")%>">		
		<span class="el-icon-circle-export el-icon" @click="exportCSV"></span>
	</div>
	<el-ctable id="activeFaultTable" ref="ctableUpsList" :time="6" :height="height" url='${ctx}/ups/queryUpsInfosList.action'
		:query-params="params_ups" pagination="true" :rownumber=true>
			<template slot="toolbar">
				<span 
					class="el-icon-operation-settings el-icon" 
					style="position:absolute;left:5px;z-index:99;width:30px;height: 40px;box-shadow:none;" 
					:style="{top:setTop}"
					@click="columnSetting"
				></span>
				<div class="queryGroup">
					<el-input class='pairgrid-query' v-model='searchText'  @keyup.enter.native="searchResult"
							placeholder="<%=rb.getString("DianYuanBianMa") %>/<%=rb.getString("IPDiZhi") %>/<%=rb.getString("UPSSiteID") %>"></el-input>
					<i @click='searchResult' class="el-icon el-icon-common-search ml10" ></i>
				</div>
				<div class="showHideItem" id="ups_column_setting">
					<div class="select-all-cls" style="padding-left: 10px;">
						<el-checkbox :indeterminate="!dragAll" v-model="dragAll" @change="dragAllChange"></el-checkbox> 
						<span><%=rb.getString("QuanXuan")%></span>
					</div>
					<el-checkbox-group v-model="dragCol">
						<draggable
							class="list-group"
							v-model="sortColumns"
							v-bind="dragOptions">
							<transition-group type="transition" :name="!drag? 'flip-list':null">
								<div v-for="(col,idx) in sortColumns" :key="col.field" class="list-group-item">
									<el-checkbox :label="col.field" :key="col.field" :disabled="col.disabled">{{col.label}} </el-checkbox>
									<span v-if="false" style="position: absolute;right: 5px;top: 5px;">{{idx+1}}</span>
								</div>
							</transition-group>
						</draggable>
					</el-checkbox-group>

					<div class="windowButtonGroup" style="float:none !important;margin:20px 0 20px 30px">			
						<a class="linkbutton linkbutton_trend" @click="ColumnConfig"><span><%=rb.getString("QueDing")%></span></a>
						<a class="linkbutton linkbutton_nowanna" @click="cancelConfig"><span><%=rb.getString("QuXiao")%></span></a>
					</div>
				</div>
			</template>
			<!-- 主列表 -->
			<el-table-column label='' width="30" prop="">
				<template slot-scope="scope">
					<div class="el-icon el-icon-operation-more curpo" @click="optClick(scope.row,event)" v-clickoutside="handerClose" ></div>
				</template>
			</el-table-column>
			
			<el-table-column prop="connection_status" sortable  width="50"  >
				<template slot-scope="scope">
					<div :class="{
						'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
						'':scope.row.have_connected==2,
						'conn_exc':scope.row.connection_status=='Exception',
						'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
				</template>
				</el-table-column>
			<el-table-column label='<%=rb.getString("GaoJingShu") %>' width="110" prop="alarm_count" sortable>
				<template slot-scope="scope">
					<div  class="txc" v-if='scope.row.alarm_serverity === 31004'>
						<span class="alarmWarning alarmListSty">
							{{scope.row.alarm_count}}
						</span>
					</div>
					<div  class="txc" v-if='scope.row.alarm_serverity === 31003'>
						<span class="alarmMinor alarmListSty">
							{{scope.row.alarm_count}}
						</span>
					</div>
					<div class="txc" v-if='scope.row.alarm_serverity === 31002'>
						<span class="alarmMajor alarmListSty">
							{{scope.row.alarm_count}}
						</span>
					</div>
					<div  class="txc" v-if='scope.row.alarm_serverity === 31001 '>
						<span class="alarmCritical alarmListSty">
							{{scope.row.alarm_count}}
						</span>
					</div>
					<div  class="txc" v-if='scope.row.alarm_serverity === null '>
							{{scope.row.alarm_count}}
					</div>
				</template>
			</el-table-column>
			<el-table-column v-for="col in columns" :prop="col.field" :label="col.label" :sortable="col.sortable" :min-width="col.width" v-if="showColums.includes(col.field)">
				<template slot-scope="scope">
					<template v-if="col.field == 'power_state'">
						<div v-if="scope.row.power_state == 1">
							<span class="el-icon el-icon-status-powerON"> <span class="f12"><%=rb.getString("GongDian") %></span> </span>
						</div>
						<div v-if="scope.row.power_state == 0">
							<span class="el-icon el-icon-status-powerOFF"><span class="f12"><%=rb.getString("DuanDian") %></span></span>
						</div>
						<div v-if="scope.row.power_state == '--'" class="texc">
							<span class="f12">--</span>
						</div>
					</template>

					<template v-if="col.field == 'uptime'">
						<div v-if="scope.row.uptime == '--'" class="texc">
							<span class="f12">--</span>
						</div>
						<div v-else>
							<span class="f12">{{scope.row.uptime}}</span>
						</div>
					</template>

					<template v-if="col.field == 'board_temperature'">
						<div v-if='scope.row.board_temperature <= 0' class="wendu-height">
							<span class="el-icon el-icon-status-temperature"><span class="f12">{{scope.row.board_temperature}}℃</span></span>
						</div>
						<div v-if='scope.row.board_temperature > 0 && scope.row.board_temperature <= 10' class="wendu-middle">
							<span class="el-icon el-icon-status-temperature"><span class="f12">{{scope.row.board_temperature}}℃</span></span>
						</div>
						<div v-if='scope.row.board_temperature > 10 && scope.row.board_temperature <= 40' class="wendu-low">
							<span class="el-icon el-icon-status-temperature"><span class="f12">{{scope.row.board_temperature}}℃</span></span>
						</div>
						<div v-if='scope.row.board_temperature > 40 && scope.row.board_temperature < 60' class="wendu-middle">
							<span class="el-icon el-icon-status-temperature"><span class="f12">{{scope.row.board_temperature}}℃</span></span>
						</div>
						<div v-if='scope.row.board_temperature >=60' class="wendu-height">
							<span class="el-icon el-icon-status-temperature"><span class="f12">{{scope.row.board_temperature}}℃</span></span>
						</div>
						<div v-if='scope.row.board_temperature==="--"' class="wendu-height texc">
							<span class="f12">--</span>
						</div>
					</template>

					<template v-if="col.field == 'dc_voltage'">
						<div v-if="scope.row.dc_voltage == '--'" class="texc">
							<span class="f12">--</span>
						</div>
						<div v-else>
							<span class="f12">{{scope.row.dc_voltage}}V</span>
						</div>
					</template>

					<template v-if="col.field == 'sfp_state'">
						<div v-if='scope.row.sfp_state == "DISCONNECT"' class="sfp-wait">
							<span class="el-icon el-icon-status-SFP" title='<%=rb.getString("WuMoKuai") %>'></span>
						</div>
						<div v-if='scope.row.sfp_state == "PLUGIN"' class="sfp-success">
							<span class="el-icon el-icon-status-SFP" title='<%=rb.getString("MoKuaiYiCha") %>'></span>
						</div>
						<div v-if='scope.row.sfp_state == "FAULT"' class="sfp-error">
							<span class="el-icon el-icon-status-SFP" title='<%=rb.getString("WangLuoGuZhang") %>'></span>
						</div>
						<div v-if='scope.row.sfp_state == "--"' class="texc">
							<span>--</span>
						</div>
					</template>

					<template v-if="col.field == 'lan_state'">
						<div v-if='scope.row.lan_state == "--"' class="texc">
							<span>--</span>
						</div>
						<div v-else>

							<div class="fl mr5 lan-success" v-if='scope.row.lan_state.lan_0_state == "UP"' ><span class="el-icon el-icon-status-LAN" title='<%=rb.getString("WangLuoYiLianJie") %>'></span></div>
							<div class="fl mr5 lan-error"v-if='scope.row.lan_state.lan_0_state ==  "FAULT"'><span class="el-icon el-icon-status-LAN"  title='<%=rb.getString("WangLuoGuZhang") %>'></span></div>
							<div class="fl mr5 lan-wait" v-if='scope.row.lan_state.lan_0_state == "DOWN"'><span class="el-icon el-icon-status-LAN"  title='<%=rb.getString("WangLuoWeiLianJie") %>'></span></div>
							<div class="fl mr5 lan-wait" v-if='scope.row.lan_state.lan_0_state == "--"'><span >--</span></div>
							
							<div class="fl mr5 lan-success" v-if='scope.row.lan_state.lan_1_state == "UP"'><span class="el-icon el-icon-status-LAN" title='<%=rb.getString("WangLuoYiLianJie") %>'></span></div>
							<div class="fl mr5 lan-error" v-if='scope.row.lan_state.lan_1_state ==  "FAULT"'><span class="el-icon el-icon-status-LAN" title='<%=rb.getString("WangLuoGuZhang") %>'></span></div>
							<div class="fl mr5 lan-wait" v-if='scope.row.lan_state.lan_1_state == "DOWN"'><span class="el-icon el-icon-status-LAN"  title='<%=rb.getString("WangLuoWeiLianJie") %>'></span></div>

							<div class="fl mr5 lan-success" v-if='scope.row.lan_state.lan_2_state == "UP"'><span class="el-icon el-icon-status-LAN" title='<%=rb.getString("WangLuoYiLianJie") %>'></span></div>
							<div class="fl mr5 lan-error" v-if='scope.row.lan_state.lan_2_state ==  "FAULT"'><span class="el-icon el-icon-status-LAN" title='<%=rb.getString("WangLuoGuZhang") %>'></span></div>
							<div class="fl mr5 lan-wait" v-if='scope.row.lan_state.lan_2_state == "DOWN"'><span class="el-icon el-icon-status-LAN"  title='<%=rb.getString("WangLuoWeiLianJie") %>'></span></div>

							<div class="fl mr5 lan-success" v-if='scope.row.lan_state.lan_3_state == "UP"'><span class="el-icon el-icon-status-LAN" title='<%=rb.getString("WangLuoYiLianJie") %>'></span></div>
							<div class="fl mr5 lan-error" v-if='scope.row.lan_state.lan_3_state ==  "FAULT"'><span class="el-icon el-icon-status-LAN" title='<%=rb.getString("WangLuoGuZhang") %>'></span></div>
							<div class="fl mr5 lan-wait" v-if='scope.row.lan_state.lan_3_state == "DOWN"'><span class="el-icon el-icon-status-LAN"  title='<%=rb.getString("WangLuoWeiLianJie") %>'></span></div>
						</div>
					</template>
					
					<template v-if="col.field == 'soc'">
						<div v-if='scope.row.soc >=0 && scope.row.soc <= 40'>
							<span class="el-icon el-icon-status-low">
								<span class="f12">{{scope.row.soc}}%</span>
							</span>
						</div>
						<div v-if='scope.row.soc >40 && scope.row.soc <= 70'>
							<span class="el-icon el-icon-status-run">
								<span class="f12">{{scope.row.soc}}%</span>
							</span>
						</div>
						<div v-if='scope.row.soc >70 && scope.row.soc <= 100'>
							<span class="el-icon el-icon-status-high">
								<span class="f12">{{scope.row.soc}}%</span>
							</span>
						</div>
						<div v-if='scope.row.soc==="--"' class="texc">	
							--
						</div>
					</template>

					<template v-if="col.field == 'battery_num'">
						<div v-if='scope.row.battery_num === "--"' class="texc">--</div>
						<div v-else>{{scope.row.battery_num}}</div>
					</template>
					<template v-if="!['power_state','uptime','board_temperature','sfp_state','lan_state','soc','battery_num','dc_voltage'].includes(col.field)">{{scope.row[col.field]}}</template>
				</template>
			</el-table-column>
	</el-ctable>
	<el-cmenu ref="menuActiveFault" :data="menus" @click="clickMenu"></el-cmenu>
	<!--弹窗页面部分 -->
	<el-slide class="upsMonitorSlideCls" ref="slide" :title="'<%=rb.getString("SheZhi")%>'" :footer="false" :header='true' position="right" 
			:height="height" :modal='modal'  :width="width" @cancel="closeSettingInfo">
			<div class="modelBox">
				<div class='el-card__body' style="padding:20px;border:none">
					<el-form  label-position="top" :model="settingsInfo" :rules="settingsRules" ref="settingsForm">
						<div class="form-group last" >
								<div class="group-title">
									<span class="title-icon"></span><span class="title-text">Basic Setting</span>
								</div>
								<div class="plr15 ml20">
									<el-form-item label="Site ID" prop="siteId">
										<el-input v-model="settingsInfo.siteId" placeholder='<%=rb.getString("QingShuRuSiteId") %>' class="w290" maxlength='48'></el-input>
									</el-form-item>
									<el-form-item label="<%=rb.getString("Title_SheBeiMingCheng") %>">
										<el-input v-model="settingsInfo.deviceName" placeholder='<%=rb.getString("QingShuRu") %><%=rb.getString("Title_SheBeiMingCheng") %>' class="w290"></el-input>
									</el-form-item>
								</div>
							</div>
					</el-form>
			</div>
			<div class='footer'>	
				<div class="linkbuttonGroup">
					<el-button type="primary" @click="updateSettings"><%=rb.getString("QueDing")%></el-button>
					<el-button @click="closeSettingInfo"><%=rb.getString("QuXiao")%></el-button>
				</div>
			</div>
			</div>
			
	</el-slide>
	<!-- 详情-->
	 <el-slide ref="slider" :url="slideUrl" :title="slideTitle" :footer="footerShow" :header='headerShow' :position="slidePosition" :modal="true" :height="sliderHeight" :width="sliderWidth" 
	 @cancel='cancelSlide'></el-slide>
	
</div>
</div>

<script>
var faultListVue = new Vue({
	el:'#falutListPage',
	data(){
		let siteIdValidator = (rule,value,cb)=>{
			let reg = /^[a-z0-9-_]{0,48}$/i;
			if(!reg.test(value)){
				cb('max length: 48,and allows enter letters,numbers,underlines,hyphen');
			}else{
				cb()
			}
		}
		return{
			searchText:'',
			height:'100%',
			menus:[],
			modal:true,
			width:'800px',
			timeZone:timeZone,
			params_ups:{ // 初始化数据参数
				searchText: '',
				timeZone: timeZone,
			},
			deviceType:[],
			rowData:[],
			dialogTitle:'',
			clearFlag:false,
			sortActive:'',
			orderActive:'',
			selectedIds:'',
			settingsInfo:{ // 设置详情页面的参数
				from:'',
				siteId: '',
				upsCode: '',
				deviceName: ''
			},
			settingsRules:{
				siteId:[
					{validator:siteIdValidator},
				]
			},
			slideUrl:'',
			slideTitle:'',
			sliderHeight: '100%',
			sliderWidth: '100%',
			footerShow: true,
			headerShow: true,
			slidePosition: 'top',

			setTop:'63px',
			showProps: ['power_state','serial_number','product','model','software_version','uptime','site_id','device_name','ipaddress','board_temperature','soc','battery_num','dc_voltage'],
			dragCol: ['power_state','serial_number','product','model','software_version','uptime','site_id','device_name','ipaddress','board_temperature','soc','battery_num','dc_voltage'],
			drag: false,
			columns: [
				{field: 'power_state', label: '<%=rb.getString("ACPpower") %>',sortable: true,disabled: true,width: 130},
				{field: 'serial_number', label: '<%=rb.getString("DianYuanBianMa") %>',sortable: true,disabled: true,width: 130},
				{field: 'product', label: '<%=rb.getString("ChangPinXingHao") %>',disabled: true,width: 110},
				{field: 'model', label: '<%=rb.getString("SheBeiXingHao") %>',disabled: true,width: 100},
				{field: 'software_version', label: '<%=rb.getString("SoftwareVersion") %>',width: 170},
				{field: 'uptime', label: '<%=rb.getString("YunXingShiJian") %>',width: 130},
				{field: 'site_id', label: '<%=rb.getString("UPSSiteID") %>',width: 110},
				{field: 'device_name', label: '<%=rb.getString("Title_SheBeiMingCheng") %>',width: 110},
				{field: 'ipaddress', label: '<%=rb.getString("IPDiZhi") %>',width: 130},
				{field: 'board_temperature', label: '<%=rb.getString("BoardTempaerature") %>',width: 150},
				{field: 'dc_voltage', label: '<%=rb.getString("DCVoltage") %>',width: 100},
				{field: 'sfp_state', label: '<%=rb.getString("SFPState") %>',width: 100},
				{field: 'lan_state', label: '<%=rb.getString("PortState") %>',width: 130},
				{field: 'soc', label: '<%=rb.getString("SOC")%>',width: 80},
				{field: 'battery_num', label: '<%=rb.getString("DianChiGeShu")%>',width: 150}
			],
			sortColumns: [
				{field: 'power_state', label: '<%=rb.getString("ACPpower") %>',sortable: true,disabled: true,width: 130},
				{field: 'serial_number', label: '<%=rb.getString("DianYuanBianMa") %>',sortable: true,disabled: true,width: 130},
				{field: 'product', label: '<%=rb.getString("ChangPinXingHao") %>',disabled: true,width: 110},
				{field: 'model', label: '<%=rb.getString("SheBeiXingHao") %>',disabled: true,width: 100},
				{field: 'software_version', label: '<%=rb.getString("SoftwareVersion") %>',width: 170},
				{field: 'uptime', label: '<%=rb.getString("YunXingShiJian") %>',width: 130},
				{field: 'site_id', label: '<%=rb.getString("UPSSiteID") %>',width: 110},
				{field: 'device_name', label: '<%=rb.getString("Title_SheBeiMingCheng") %>',width: 110},
				{field: 'ipaddress', label: '<%=rb.getString("IPDiZhi") %>',width: 130},
				{field: 'board_temperature', label: '<%=rb.getString("BoardTempaerature") %>',width: 150},
				{field: 'dc_voltage', label: '<%=rb.getString("DCVoltage") %>',width: 100},
				{field: 'sfp_state', label: '<%=rb.getString("SFPState") %>',width: 100},
				{field: 'lan_state', label: '<%=rb.getString("PortState") %>',width: 130},
				{field: 'soc', label: '<%=rb.getString("SOC")%>',width: 80},
				{field: 'battery_num', label: '<%=rb.getString("DianChiGeShu")%>',width: 150}
			],
			origionSort: [],
		}
		
	},
	computed: {
		showColums() {
			var vm = this,
				columns = vm.columns,
				props = columns.map(function(col){
					return col.field;
				});

			props = props.filter(function(code){
				return vm.showProps.includes(code);
			});

			return props;
		},
		dragOptions() {

			return {
				animation: 200,
				group: 'description',
				disabled: false,
				ghostClass: 'ghost'
			};
		},
		dragAll() {
			var vm = this;
			return vm.dragCol.length == vm.columns.length;
		}
	},
	methods:{
		dragAllChange(val) {
			var vm = this,
				fields = vm.columns.map(function(item){
					return item.field;
				}),
				filters = vm.columns.filter(function(item){
					return item.disabled;
				}).map(function(item){
					return item.field;
				});
			
			vm.dragCol = val?fields:filters;
		},
		// 表格展示设置 保存
		ColumnConfig() {
			var vm = this,
				url = '${ctx}/system/column/setting/insert.action',
				sortCol = [];

			vm.sortColumns.map(function(item){
				sortCol.push(item.field);
			});

			var params = {
					pageName: '5',
					showColumn: vm.dragCol.join(','),
					sortColumn: sortCol.join(',')	
				};

			axios.post(url, params).then(function(res){
				vm.columns = [];
				vm.sortColumns.map(function(item){
					vm.columns.push(Object.assign({}, item));
				});

				vm.showProps = [];
				vm.dragCol.map(function(code){
					vm.showProps.push(code);
				});

				vm.close();
			});
		},
		// 取消保存
		cancelConfig() {
			var vm = this;

			vm.dragCol = [];
			vm.showProps.map(function(code){
				vm.dragCol.push(code);
			});

			vm.close();
		},
		close() {
			$("#ups_column_setting").slideUp(500);
		},
		// 表格展示设置打开
		columnSetting() {
			$("#ups_column_setting").slideDown();
		},
		// 搜索函数
		searchResult(){
			let vm = this

			this.params_ups.searchText = vm.searchText;
			this.$refs["ctableUpsList"].refresh();	 
		},
		handerClose(){ //点击页面其他地方菜单收起
			this.$refs.menuActiveFault.hide();
		},
		optClick(row,ev){ // 操作项： 1.设置 2.电池详情
			var vm = this,
				clearFlag = false;   //清除状态 
				
			var upsEnable = true;
			$.ajax({
				type:'POST',
				url:'${ctx}/ups/getUPSOperationItem.action',
				data:{upsCode: row.ups_code},
				async:false,
				dataType:'json',
				success:function(data){
					if(data) {
						upsEnable = data.upsFlag == true;
					}
				}
			});

			vm.menus= [
				{label:'<%=rb.getString("XinXi")%>',cls:"el-icon el-icon-operation-info",code:'info'},
				{label:'<%=rb.getString("SheZhi")%>',cls:"el-icon el-icon-operation-settings CODE_UPS hidden",code:'settings', show: upsEnable},
				{label:'<%=rb.getString("ChongQi")%>',cls:"el-icon el-icon-operation-reboot CODE_UPS hidden",code:'reboot', show: upsEnable},
					
			]
			this.rowData = row
			vm.$nextTick(function(){
				document.body.click();
				vm.$refs.menuActiveFault.show(ev);
				
			});
		},
		clickMenu(ev){ //操作项点击方法 
			var codes = {
				settings:this.settingInfo,
				info: this.viewInfo, //跳转到 信息 函数
				reboot:this.rebootInfo
			}
			if(codes[ev.code]){
				codes[ev.code](this.$root.rowData)
			}
		},
		settingInfo(){ // 设置弹窗
			var vm = this,
				type,
				rowData = this.rowData;	
			
			vm.settingsInfo.siteId = rowData.site_id ,
			vm.settingsInfo.upsCode = rowData.ups_code,
			vm.settingsInfo.deviceName = rowData.device_name
			vm.$refs.slide.showSlide(function(){
				vm.modal = false;
				eventBus.$emit('detail-info',type,rowData.site_id);
			});
	
		},
		viewInfo(){ // 查看详情
			var vm = this;
			vm.slideUrl = '${ctx}/ups/goUPSInformationPage.action';
			vm.slideTitle = '<%=rb.getString("XinXi")%>';
			vm.slidePosition = 'left';
			vm.sliderHeight = '100%';
			vm.footerShow = false;
			vm.headerShow = true;
			vm.$refs.slider.showSlide(function(){
				eventBus.$emit('open-dialog',vm.rowData);
			});
		},
		// 重启
		rebootInfo(code){
			var vm = this;
			let params = {};
			var url = "${ctx}/ups/reboot.action"
			params.ups_code = code.ups_code;
		
			vm.$confirm('<%=rb.getString("QueDingChongQiSheBei")%>','<%=rb.getString("QueRen")%>',{
				customClass:"warningConfirm",
				confirmButtonText:'<%=rb.getString("QueDing")%>', 
				cancalButtonText:'<%=rb.getString("QuXiao")%>',
				type:'warning',
				closeOnClickModal:false
			}).then(()=>{
				vm.$message({
						message:  '<%=rb.getString("MingLingYiXiaFa")%>',
						type:'success',
					});	
				axios.post(url,stringify(params)).then(function(response){
					
					
				}).catch(function(error){})
				
			}).catch(()=>{
				
			})
		},
	
		cancelSlide(){
			this.$refs.slider.hide();
		},
		closeSettingInfo(){   //关闭设置页面  
			this.$refs.slide.hide();
		},
	
		exportCSV(){ // 导出文件
			var vm = this,
				params={
                    timeZone: timeZone,
                    searchText: vm.params_ups.searchText
                },
				exportUrl ="${ctx}/ups/exportUpsToCSV.action";
	
			exportByForm(exportUrl,params);
		},
	
		
		// 参数设置确定操作函数
		updateSettings(){
			var vm = this,
				url= '${ctx}/ups/updateUpsInfos.action',
				obj= {
					siteId:vm.settingsInfo.siteId,
					upsCode:vm.settingsInfo.upsCode,
					deviceName:vm.settingsInfo.deviceName
				};
				vm.$refs.settingsForm.validate((r)=>{
					if(r){
						axios.post(url, stringify(obj)).then(function(res){
							if(res.data.success){
								vm.$message({
									message: '<%=rb.getString("ChengGong")%>',
									type:'success',
								});
								vm.closeSettingInfo()
								vm.$refs.ctableUpsList.refresh();
							}else {
								vm.$message({
									message: res.data.message,
									type:'error',
								});
							}
						});
					}else{
						vm.$message({
							message: 'max length: 48,and allows enter letters,numbers,underlines,hyphen',
							type:'error',
						});	
					}
				})
				
		},	
	},
	created() {
		var vm = this;

		axios.post('${ctx}/system/column/setting/load/5').then(function(res){
			var data = res.data.data,
				sortCodes = (data.sortColumn||'').split(','),
				sortList = vm.columns;

			(data.showColumn||'').split(',').map(function(code){
				if(!vm.showProps.includes(code)) {
					vm.showProps.push(code);
				}
				if(!vm.dragCol.includes(code)) {
					vm.dragCol.push(code);
				}
			});
			vm.origionSort = sortCodes;

			vm.columns = sortList.sort(function(n, m) {
				var idxn = sortCodes.indexOf(n.field)==-1?100:sortCodes.indexOf(n.field),
					idxm = sortCodes.indexOf(m.field)==-1?100:sortCodes.indexOf(m.field);

				return idxn - idxm;
			});
			vm.sortColumns = sortList.sort(function(n, m) {
				var idxn = sortCodes.indexOf(n.field)==-1?100:sortCodes.indexOf(n.field),
					idxm = sortCodes.indexOf(m.field)==-1?100:sortCodes.indexOf(m.field);

				return idxn - idxm;
			});
		});
	},
	mounted(){

	}
});
</script>