<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style type="text/css">
#gnbSystemPage{
	height: 100%;
	width: 100%;
}
#gnbSystemPage .itemMainBoxCls{
	border-radius:10px;
	background:#fff;
	height:100%;
	width: 100%;
    display: flex;
    flex-direction: column;
    position: relative;
}
#gnbSystemPage .itemMainBoxTitle {
	height:36px;
	padding-left: 20px;
    line-height: 36px;
	font-size:14px;
	font-weight:bold;
	border-bottom: 1px solid #E9E9E9;
}
#gnbSystemPage .itemMainBoxCenter{
	width: 100%;
	flex:1;
	overflow: auto;
}
#gnbSystemPage .itemMainBoxFooter{
    display: flex;
    align-items: center;
    border-top : 1px solid #E9E9E9;
	height:48px;
	background-color: #FFFFFF;
    box-sizing: border-box;
    width: 100%;
	padding-left: 20px;
}
#gnbSystemPage .rightContentCls .contentTableTitle{
	display: flex;
	justify-content: space-between;
	font-weight: 550;
	width: 100%;
}
#gnbSystemPage .rightContentCls .contentTableTitle>div:nth-child(1){
	font-size: 12px;
}
#gnbSystemPage .moreIpItemBoxCls{
	display: flex;
	flex-wrap: wrap;
	width: 100%;
}
#gnbSystemPage .leftAndRightItemCls{
	width:40%;
	min-width:400px;
	margin-bottom: 20px;
}
#gnbSystemPage .itemListBoxCls{
	padding-top: 5px;
}
#gnbSystemPage .itemCls{
	height: 24px;
	display: inline-block;
	line-height: 24px;
	border: 1px solid #4D84FF;
	box-sizing: border-box;
	padding: 0px 10px;
	margin-right: 10px;
	margin-bottom: 10px;
}
#gnbSystemPage .itemListBoxCls .el-icon-close{
	font-size: unset;
	position: unset;
	top: unset;
	right: unset;
}
#gnbSystemPage .leftAndRightItemCls .el-input__suffix{
	height: 26px;
	display: flex;
	align-items: center;
}
#gnbSystemPage .el-form-item{
	margin-bottom: 20px;
}
#gnbSystemPage .errorBoxCls{
	color:red;
	font-size:10px;
}
#gnbSystemPage .multiPlmnEnableBoxCls .el-form-item__label{
	padding-top: 13px;
	margin-right: 20px;
}
#gnbSystemPage .el-form-item .el-form-item__label{
	font-size: 12px;
}
</style>

<div id="gnbSystemPage">
	<div class="itemMainBoxCls">
		<div class="itemMainBoxTitle">
			System
			<!-- 按钮  同步 -->
			<div class="newIconBoxCls-bt" style="right:20px;top:5px;" @click="syncSettingsClick" tip="<%=rb.getString("TongBu")%>">
				<span class="el-icon el-icon-circle-refresh"></span>
			</div>
		</div>
		<div class="itemMainBoxCenter">
			<el-form :model='ruleForm' ref="ruleForm" :rules="rules" label-position="top">
				<el-collapse v-model="activeCollapse">
					<el-collapse-item name="L3">
						<template slot='title'>
							<p style="display:inline-block;margin-left:40px;">
								<span style="font-size:14px;font-weight:bold">L3 Log Setting</span>
							</p>
						</template>
						<div class="rightContentCls" >
							<div style="display:flex;margin-left:16px;flex-wrap: wrap">
								<el-form-item prop='L3_RRCLogLevel' style="width:40%;min-width:400px;" label="RRC Log Level" label-width="160px">
                                    <el-select v-model='ruleForm.L3_RRCLogLevel'>
										<el-option label='Fatal' value='1'></el-option>
										<el-option label='Error' value='3'></el-option>
										<el-option label='Warning' value='7'></el-option>
                                        <el-option label='Information' value='15'></el-option>
										<el-option label='Brief' value='31'></el-option>
										<el-option label='Detailed' value='63'></el-option>
                                        <el-option label='Detailed All' value='127'></el-option>
									</el-select>
								</el-form-item>
                                <el-form-item prop='L3_RRMLogLevel' style="width:40%;min-width:400px;" label="RRM Log Level" label-width="160px">
                                    <el-select v-model='ruleForm.L3_RRMLogLevel'>
										<el-option label='Fatal' value='1'></el-option>
										<el-option label='Error' value='3'></el-option>
										<el-option label='Warning' value='7'></el-option>
                                        <el-option label='Information' value='15'></el-option>
										<el-option label='Brief' value='31'></el-option>
										<el-option label='Detailed' value='63'></el-option>
                                        <el-option label='Detailed All' value='127'></el-option>
									</el-select>
								</el-form-item>
                                <el-form-item prop='L3_DUMGRLogLevel' style="width:40%;min-width:400px;" label="DUMGR Log Level" label-width="160px">
                                    <el-select v-model='ruleForm.L3_DUMGRLogLevel'>
										<el-option label='Fatal' value='1'></el-option>
										<el-option label='Error' value='3'></el-option>
										<el-option label='Warning' value='7'></el-option>
                                        <el-option label='Information' value='15'></el-option>
										<el-option label='Brief' value='31'></el-option>
										<el-option label='Detailed' value='63'></el-option>
                                        <el-option label='Detailed All' value='127'></el-option>
									</el-select>
								</el-form-item>
								<el-form-item prop='L3_RRCLogFileSize' style="width:40%;min-width:400px;" label="RRC Log File Size" label-width="140px" class='validate-item'>
									<el-input v-model.trim='ruleForm.L3_RRCLogFileSize'>
										<template slot="append"><%=rb.getString("FanWei")%>：3~255,Integer</template>
									</el-input>
								</el-form-item>
                                <el-form-item prop='L3_RRMLogFileSize' style="width:40%;min-width:400px;" label="RRM Log File Size" label-width="140px" class='validate-item'>
									<el-input v-model.trim='ruleForm.L3_RRMLogFileSize'>
										<template slot="append"><%=rb.getString("FanWei")%>：3~255,Integer</template>
									</el-input>
								</el-form-item>
                                <el-form-item prop='L3_DUOAMOrDUMGRFileSize' style="width:40%;min-width:400px;" label="DUOAM&DUMGR Log File Size" label-width="140px" class='validate-item'>
									<el-input v-model.trim='ruleForm.L3_DUOAMOrDUMGRFileSize'>
										<template slot="append"><%=rb.getString("FanWei")%>：3~255,Integer</template>
									</el-input>
								</el-form-item>
                                <el-form-item prop='L3_RRCLogFileCount' style="width:40%;min-width:400px;" label="RRC Log File Count" label-width="140px" class='validate-item'>
									<el-input v-model.trim='ruleForm.L3_RRCLogFileCount'>
										<template slot="append"><%=rb.getString("FanWei")%>：1~30,Integer</template>
									</el-input>
								</el-form-item>
                                <el-form-item prop='L3_RRMLogFileCount' style="width:40%;min-width:400px;" label="RRM Log File Count" label-width="140px" class='validate-item'>
									<el-input v-model.trim='ruleForm.L3_RRMLogFileCount'>
										<template slot="append"><%=rb.getString("FanWei")%>：1~30,Integer</template>
									</el-input>
								</el-form-item>
                                <el-form-item prop='L3_DUOAMOrDUMGRFileCount' style="width:40%;min-width:400px;" label="DUOAM&DUMGR Log File Count" label-width="140px" class='validate-item'>
									<el-input v-model.trim='ruleForm.L3_DUOAMOrDUMGRFileCount'>
										<template slot="append"><%=rb.getString("FanWei")%>：1~30,Integer</template>
									</el-input>
								</el-form-item>
                                <el-form-item prop='L3_RRCSharedMemorySize' style="width:40%;min-width:400px;" label="RRC Shared Memory Size" label-width="140px" class='validate-item'>
									<el-input v-model.trim='ruleForm.L3_RRCSharedMemorySize'>
										<template slot="append"><%=rb.getString("FanWei")%>：3~255,Integer</template>
									</el-input>
								</el-form-item>
                                <el-form-item prop='L3_RRMSharedMemorySize' style="width:40%;min-width:400px;" label="RRM Shared Memory Size" label-width="140px" class='validate-item'>
									<el-input v-model.trim='ruleForm.L3_RRMSharedMemorySize'>
										<template slot="append"><%=rb.getString("FanWei")%>：3~255,Integer</template>
									</el-input>
								</el-form-item>
                                <el-form-item prop='L3_DUOAMOrDUMGRSharedMemorySize' style="width:40%;min-width:400px;" label="DUOAM&DUMGR Shared Memory Size" label-width="140px" class='validate-item'>
									<el-input v-model.trim='ruleForm.L3_DUOAMOrDUMGRSharedMemorySize'>
										<template slot="append"><%=rb.getString("FanWei")%>：3~255,Integer</template>
									</el-input>
								</el-form-item>
							</div>
						</div>
					</el-collapse-item>
					<el-collapse-item name="PDCP">
						<template slot='title'>
							<p style="display:inline-block;margin-left:40px;">
								<span style="font-size:14px;font-weight:bold">PDCP Log Setting</span>
							</p>
						</template>
						<div class="rightContentCls" >
							<div style="display:flex;margin-left:16px;flex-wrap: wrap">
								<el-form-item prop='PDCP_LogLevel' style="width:40%;min-width:400px;" label="PDCP Log Level" label-width="160px">
                                    <el-select v-model='ruleForm.PDCP_LogLevel'>
										<el-option label='Fatal' value='1'></el-option>
										<el-option label='Error' value='3'></el-option>
										<el-option label='Warning' value='7'></el-option>
                                        <el-option label='Information' value='15'></el-option>
										<el-option label='Brief' value='31'></el-option>
										<el-option label='Detailed' value='63'></el-option>
                                        <el-option label='Detailed All' value='127'></el-option>
									</el-select>
								</el-form-item>
                                <el-form-item prop='PDCP_NGUOrCUF1ULogLevel' style="width:40%;min-width:400px;" label="NGU/CUF1U Log Level" label-width="160px">
                                    <el-select v-model='ruleForm.PDCP_NGUOrCUF1ULogLevel'>
										<el-option label='Fatal' value='1'></el-option>
										<el-option label='Error' value='3'></el-option>
										<el-option label='Warning' value='7'></el-option>
                                        <el-option label='Information' value='15'></el-option>
										<el-option label='Brief' value='31'></el-option>
										<el-option label='Detailed' value='63'></el-option>
                                        <el-option label='Detailed All' value='127'></el-option>
									</el-select>
								</el-form-item>
							</div>
						</div>
					</el-collapse-item>
                    <el-collapse-item name="RLC">
						<template slot='title'>
							<p style="display:inline-block;margin-left:40px;">
								<span style="font-size:14px;font-weight:bold">RLC Log Setting</span>
							</p>
						</template>
						<div class="rightContentCls" >
							<div style="display:flex;margin-left:16px;flex-wrap: wrap">
								<el-form-item prop='RLC_LogLevel' style="width:40%;min-width:400px;" label="RLC Log Level" label-width="160px">
                                    <el-select v-model='ruleForm.RLC_LogLevel'>
										<el-option label='Fatal' value='1'></el-option>
										<el-option label='Error' value='2'></el-option>
										<el-option label='Warning' value='4'></el-option>
                                        <el-option label='Information' value='8'></el-option>
										<el-option label='Brief' value='16'></el-option>
										<el-option label='Detailed' value='32'></el-option>
                                        <el-option label='Detailed All' value='64'></el-option>
									</el-select>
								</el-form-item>
                                <el-form-item prop='RLC_DUF1ULogLevel' style="width:40%;min-width:400px;" label="DUF1U Log Level" label-width="160px">
                                    <el-select v-model='ruleForm.RLC_DUF1ULogLevel'>
										<el-option label='Fatal' value='1'></el-option>
										<el-option label='Error' value='3'></el-option>
										<el-option label='Warning' value='7'></el-option>
                                        <el-option label='Information' value='15'></el-option>
										<el-option label='Brief' value='31'></el-option>
										<el-option label='Detailed' value='63'></el-option>
                                        <el-option label='Detailed All' value='127'></el-option>
									</el-select>
								</el-form-item>
                                <el-form-item prop='RLC_RLCLLogLevel' style="width:40%;min-width:400px;" label="RLCL Log Level" label-width="160px">
                                    <el-select v-model='ruleForm.RLC_RLCLLogLevel'>
										<el-option label='Fatal' value='1'></el-option>
										<el-option label='Error' value='2'></el-option>
										<el-option label='Warning' value='4'></el-option>
                                        <el-option label='Information' value='8'></el-option>
										<el-option label='Brief' value='16'></el-option>
										<el-option label='Detailed' value='32'></el-option>
                                        <el-option label='Detailed All' value='64'></el-option>
									</el-select>
								</el-form-item>
							</div>
						</div>
					</el-collapse-item>
                    <el-collapse-item name="MAC">
						<template slot='title'>
							<p style="display:inline-block;margin-left:40px;">
								<span style="font-size:14px;font-weight:bold">MAC Log Setting</span>
							</p>
						</template>
						<div class="rightContentCls" >
							<div style="display:flex;margin-left:16px;flex-wrap: wrap">
								<el-form-item prop='MAC_LogLevel' style="width:40%;min-width:400px;" label="MAC Log Level" label-width="160px">
                                    <el-select v-model='ruleForm.MAC_LogLevel'>
										<el-option label='Fatal' value='1'></el-option>
										<el-option label='Error' value='2'></el-option>
										<el-option label='Warning' value='4'></el-option>
                                        <el-option label='Information' value='8'></el-option>
										<el-option label='Brief' value='16'></el-option>
										<el-option label='Detailed' value='32'></el-option>
                                        <el-option label='Detailed All' value='64'></el-option>
									</el-select>
								</el-form-item>
                                <el-form-item prop='MAC_SCHLogLevel' style="width:40%;min-width:400px;" label="SCH Log Level" label-width="160px">
                                    <el-select v-model='ruleForm.MAC_SCHLogLevel'>
										<el-option label='Fatal' value='1'></el-option>
										<el-option label='Error' value='2'></el-option>
										<el-option label='Warning' value='4'></el-option>
                                        <el-option label='Information' value='8'></el-option>
										<el-option label='Brief' value='16'></el-option>
										<el-option label='Detailed' value='32'></el-option>
                                        <el-option label='Detailed All' value='64'></el-option>
									</el-select>
								</el-form-item>
							</div>
						</div>
					</el-collapse-item>
                    <el-collapse-item name="OAM">
						<template slot='title'>
							<p style="display:inline-block;margin-left:40px;">
								<span style="font-size:14px;font-weight:bold">OAM Log Setting</span>
							</p>
						</template>
						<div class="rightContentCls" >
							<div style="display:flex;margin-left:16px;flex-wrap: wrap">
                                <el-form-item prop='OAM_LogFileCount' style="width:40%;min-width:400px;" label="OAM Log File Count" label-width="140px" class='validate-item'>
									<el-input v-model.trim='ruleForm.OAM_LogFileCount'>
										<template slot="append"><%=rb.getString("FanWei")%>：1~10,Integer</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='OAM_SharedMemorySize' style="width:40%;min-width:400px;" label="OAM Shared Memory Size" label-width="140px" class='validate-item'>
									<el-input v-model.trim='ruleForm.OAM_SharedMemorySize'>
										<template slot="append"><%=rb.getString("FanWei")%>：3~255,Integer</template>
									</el-input>
								</el-form-item>
                                <el-form-item prop='OAM_LogFileSize' style="width:40%;min-width:400px;" label="OAM Log File Size" label-width="140px" class='validate-item'>
									<el-input v-model.trim='ruleForm.OAM_LogFileSize'>
										<template slot="append"><%=rb.getString("FanWei")%>：3~255,Integer</template>
									</el-input>
								</el-form-item>
							</div>
						</div>
					</el-collapse-item>
					
			</el-form>
		</div>
		<div class='itemMainBoxFooter'>
			<el-button type="primary" @click="settingsSubmit"><%=rb.getString("QueDing")%></el-button>
			<el-button @click="closeSettings" ><%=rb.getString("QuXiao")%></el-button>
		</div>
	</div>
</div>

<script>
var regIp = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/,
	regKey = /^[A-Fa-f0-9]{32}$/,
	regNumber = /^[0-9]{15}$/;
var gnbSystemPage = new Vue({
	el: '#gnbSystemPage', 
	data() {
		var vm = this,
			validateRange = (rule,value,callback)=>{
				var min = rule.min;
				var max = rule.max;
				var mag = rule.mag;
				var isRequired = rule.isRequired;
				var reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/;

				if(value == '' || value == undefined || value == null){
					if(isRequired){
						callback(new Error(mag))
					}else{
						callback();
					}
				}else{
					if(reg.test(value) && value >= min && value <= max){
						callback();
					}else{
						callback(new Error(mag))
					}
				}
			};
		return {
			activeCollapse:['L3','PDCP','RLC','MAC','OAM'],
			rowDataInfo: [],
            smallCellCode:'',
			ruleForm:{
				L3_RRCLogLevel:'',
                L3_RRMLogLevel:'',
                L3_DUMGRLogLevel:'',
                L3_RRCLogFileSize:'',
                L3_RRMLogFileSize:'',
                L3_DUOAMOrDUMGRFileSize:'',
                L3_RRCLogFileCount:'',
                L3_RRMLogFileCount:'',
                L3_DUOAMOrDUMGRFileCount:'',
                L3_RRCSharedMemorySize:'',
                L3_RRMSharedMemorySize:'',
                L3_DUOAMOrDUMGRSharedMemorySize:'',

                PDCP_LogLevel:'',
                PDCP_NGUOrCUF1ULogLevel:'',

                RLC_LogLevel:'',
                RLC_DUF1ULogLevel:'',
                RLC_RLCLLogLevel:'',

                MAC_LogLevel:'',
                MAC_SCHLogLevel:'',

                OAM_LogFileCount:'',
                OAM_SharedMemorySize:'',
                OAM_LogFileSize:'',
			},
			rules:{
				L3_RRCLogFileSize:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:3,max:255,isRequired:true,mag:'<%=rb.getString("FanWei")%>：3~255,Integer'}
				],
                L3_RRMLogFileSize:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:3,max:255,isRequired:true,mag:'<%=rb.getString("FanWei")%>：3~255,Integer'}
				],
                L3_DUOAMOrDUMGRFileSize:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:3,max:255,isRequired:true,mag:'<%=rb.getString("FanWei")%>：3~255,Integer'}
				],
                L3_RRCLogFileCount:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:1,max:30,isRequired:true,mag:'<%=rb.getString("FanWei")%>：1~30,Integer'}
				],
                L3_RRMLogFileCount:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:1,max:30,isRequired:true,mag:'<%=rb.getString("FanWei")%>：1~30,Integer'}
				],
                L3_DUOAMOrDUMGRFileCount:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:1,max:30,isRequired:true,mag:'<%=rb.getString("FanWei")%>：1~30,Integer'}
				],
                L3_RRCSharedMemorySize:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:3,max:255,isRequired:true,mag:'<%=rb.getString("FanWei")%>：3~255,Integer'}
				],
                L3_RRMSharedMemorySize:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:3,max:255,isRequired:true,mag:'<%=rb.getString("FanWei")%>：3~255,Integer'}
				],
                L3_DUOAMOrDUMGRSharedMemorySize:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:3,max:255,isRequired:true,mag:'<%=rb.getString("FanWei")%>：3~255,Integer'}
				],

                OAM_LogFileCount:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:1,max:10,isRequired:true,mag:'<%=rb.getString("FanWei")%>：1~10,Integer'}
				],
                OAM_SharedMemorySize:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:3,max:255,isRequired:true,mag:'<%=rb.getString("FanWei")%>：3~255,Integer'}
				],
                OAM_LogFileSize:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:3,max:255,isRequired:true,mag:'<%=rb.getString("FanWei")%>：3~255,Integer'}
				],
				
			},
			casts:{
				'19DB31674F10AC064D8A7C8543F98625':'L3_RRCLogLevel',
				'893C4A58A8EA5234DCE6A1B1145516FB':'L3_RRMLogLevel',
				'879D5CE9CFB9031505319B9F4336E569':'L3_DUMGRLogLevel',
				'A1C8849986F021B618DC23A210E6042E':'L3_RRCLogFileSize',
				'16BEDA7BC1C448CD9C65AA4793C86371':'L3_RRMLogFileSize',
                'A66C61C0E1A90945150B0E1E7D08FCF5':'L3_DUOAMOrDUMGRFileSize',
				'C8386A5721A9F4C2CE3605930CE4BDCF':'L3_RRCLogFileCount',
				'7BEA8DAA1FFC15ED870657B7BD964A90':'L3_RRMLogFileCount',
				'A3D9BB9A304665BA6C0E78888CE94616':'L3_DUOAMOrDUMGRFileCount',
				'2D999C561C38072886FF0584F59E3291':'L3_RRCSharedMemorySize',
                '354BDCDC00AF4021A515E46DEB878E98':'L3_RRMSharedMemorySize',
				'7D98A8E687AFA65379CC61439DAE9919':'L3_DUOAMOrDUMGRSharedMemorySize',

                'BFD3211AF5B0EB1E85584DBB75BC1DA7':'PDCP_LogLevel',
				'2080A53737E328B32C3152C098528CEC':'PDCP_NGUOrCUF1ULogLevel',

				'E0332C61B3F7FCA1A1BAE26800495502':'RLC_LogLevel',
				'17C2129A7504BD532D4B771AD5B46A0C':'RLC_DUF1ULogLevel',
				'97B03D44A984411AACC7B2F821722B20':'RLC_RLCLLogLevel',

                '83522317CBA427C5D79CAE2CE468853D':'MAC_LogLevel',
				'3C736EF51077196F16C068409145799C':'MAC_SCHLogLevel',

                'BCF55D13445C901DC0851200F07751E0':'OAM_LogFileCount',
				'1280E1F758938214CD01606A54D2C43D':'OAM_SharedMemorySize',
                'DBD2A69CED70CB4BF0E083CCB75AFFE0':'OAM_LogFileSize',
			},
		};
	},
	computed: {
		
	},
	methods: {
		init(row){
			var vm = this;
			vm.rowDataInfo = row;
            vm.smallCellCode = row.small_cell_code;
			var codeList=[];
			Object.keys(vm.casts).forEach(function(key){
				codeList.push(key)
			});
            vm.codeList = codeList;
			vm.getParamData(vm.smallCellCode,'23005');
		},
        getParamData(code,id) {
			var vm = this,
				codes = [],
				url = '${ctx}/cell/quicksettings/getParamNodeTreeAndData.action',
				params = {
					id: id,
					cellIndex:'1',
					smallCellCode: code
				};
			axios.post(url, stringify(params)).then(function(res){
				var data = res.data;
				vm.resetFormData();
				if(data && Array.isArray(data)) {
					data.map(function(item){
						item.groups.map(function(group){
							group.list.map(function(m){
								if(m.type == 'list'){
									vm.initTable(m.url,m.label);
								}else{
									codes.push(m.name);
									// 执行赋值
									vm.setValue(m);
								}
							});
						});
					});
					initForm(vm.$refs.ruleForm);
				}
			});
		},
		// 映射赋值
		setValue(item) {
			var vm = this,
			code = item.name,
			value = item.value;

			// indexs是否含有
			var key = vm.casts[code];
			try{
				if(key){
					vm.ruleForm[key] = value;
				}
			}catch(e){}
		},
		initTable(url,type){
			var vm = this,codes = [];
			var params = {
					smallCellCode : vm.smallCellCode
			}
			axios.post(url,stringify(params)).then(res=>{
				var data = res.data;
				if(data.rows){
					data.rows.map(item=>{
						var obj = {};
						for(var key in item){
							codes.push(key);
							obj[vm.casts[key]] = item[key]
						}
					})
					initForm(vm.$refs.ruleForm);
				}
			})
		},
		// 重置form数据
		resetFormData(){
			var vm =this;
				params={
					L3_RRCLogLevel:'',
                    L3_RRMLogLevel:'',
                    L3_DUMGRLogLevel:'',
                    L3_RRCLogFileSize:'',
                    L3_RRMLogFileSize:'',
                    L3_DUOAMOrDUMGRFileSize:'',
                    L3_RRCLogFileCount:'',
                    L3_RRMLogFileCount:'',
                    L3_DUOAMOrDUMGRFileCount:'',
                    L3_RRCSharedMemorySize:'',
                    L3_RRMSharedMemorySize:'',
                    L3_DUOAMOrDUMGRSharedMemorySize:'',

                    PDCP_LogLevel:'',
                    PDCP_NGUOrCUF1ULogLevel:'',

                    RLC_LogLevel:'',
                    RLC_DUF1ULogLevel:'',
                    RLC_RLCLLogLevel:'',

                    MAC_LogLevel:'',
                    MAC_SCHLogLevel:'',

                    OAM_LogFileCount:'',
                    OAM_SharedMemorySize:'',
                    OAM_LogFileSize:'',
				};
			Object.assign(vm.ruleForm,params);
		},
		// 判断是否为空
		isNull(val){
			if(val==undefined || val == null || val =="") return true;
			else return false;
		},
		// 验证输入的是否是整数
		isInteger(str) {
			if(str.length==0){
				return false;
			}
			var reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/;
			if(!reg.test(str)){
				return false;
			}
			return true;  
		},
		tableRowClassName({row,rowIndex}){
			if(row.operateType && row.operateType == 'remove'){
				return 'hidden-row'
			}
			return ''
		},
		settingsSubmit(){
			var vm = this;
			var params = {},
				isChanged = isFormChanged(vm.$refs.ruleForm),
				isSync = false;

			if(!isChanged){
				showMsg('prompt_msg','<%=rb.getString("CanShuZhiMeiYouBianHua")%>');
				return;
			}

			vm.$refs.ruleForm.fields.map(function(field){
				var key = vm.getNameByProp(field.prop);

				if(Array.isArray(field.fieldValue)){
					var vList = field.fieldValue.map(function(item){return item}),
						oList = (field.reinitialValue||[]).map(function(item){return item}),
						val = JSON.stringify(vList.sort()),
						orVal = JSON.stringify(oList.sort());

					if(val != orVal) {
						var editList=[],subList=[];
						vList.map((items)=>{
							if(items.operateType){
								editList.push(items)
							}
						})
						editList.map((items)=>{
							if(items.operateType == 'add'){
								Object.keys(items).map((key)=>{
									if(key.slice(-3) == 'idx'){
										delete items[key]
									}
								})
							}
						})
						editList.map((items)=>{
							var objs={};
							for(var listVal in items){
								var listKey = vm.getNameByProp(listVal);
								objs[listKey] = items[listVal]
							}
							objs.cellIndex = '1';
							subList.push(objs)
						})
						
						params[key] = subList;
					};
				}else{
					if(vm.isNull(field.fieldValue) && vm.isNull(field.reinitialValue)){
						
					}else if(field.fieldValue != field.reinitialValue) {
						var editData={
								cellIndex:'1',
								value:field.fieldValue
							}
						params[key] = editData;
					};
				}
			});
			vm.$refs.ruleForm.validate(function(valid){
				if(valid) {
					var rowCode = vm.smallCellCode,
						url = '${ctx}/cell/quicksettings/saveParamValue.action?smallCellCode='+rowCode;
					$('#gnbSetting_main').addClass('loading');
					axios.post(url,stringify({"params": JSON.stringify(params)})).then(res=>{
						var data = res.data;
						if(data["success"]){
							vm.$message.success({type:'success',message:'<%=rb.getString("ChengGong")%>'});
							eventBus.$emit('close-gnb-settingPage');
						}else{
							vm.$message.error(data["message"])
						}
						$('#gnbSetting_main').removeClass('loading');
					})
				}
			});
		},
		getNameByProp(prop) {
			var vm = this,
				reg = /^\w*\.\d*\.\w*$/
				key = prop;
			
			if(reg.test(prop)) {
				var mReg = /\.(\d*)\./,
					sufReg = /\.(\w*)$/,
					idx = prop.match(mReg)[1],
					sufStr = prop.match(sufReg)[1];

				vm.codeList.map(function(name){
					var index = vm.indexs[name];
					if(vm.casts[name] == sufStr && index == idx) {
						key = name;
					}
				});
			}else {
				vm.codeList.map(function(name){
					if(vm.casts[name] == prop) {
						key = name;
					}
				});
			}

			return key;
		},
		createId(idVal,list){
			var vm = this,
				val = idVal + '';
			if(list.includes(val) == true){
				idVal += 1 ;
				return vm.createId(idVal,list);
			}else{
				return  idVal + '';
			}
		},
		closeSettings(){
            eventBus.$emit('close-gnb-settingPage');
		},
		syncSettingsClick(){
			var vm = this,
				urls='${ctx}/cell/quicksettings/sync.action',
				params = {
					smallCellCode:vm.smallCellCode
				},
				str = Math.random().toString();
				
			axios.post(urls,stringify(params)).then(res=>{
				var data = res.data;
				if(data["success"]){
					gnbTabSettingVue.changeMain('system');
				}else{
					vm.$message.error(data["message"])
				}
			})
		}
	},
	mounted() {
		eventBus.$off("gnb-data").$on("gnb-data",this.init)
	}
});

</script>
