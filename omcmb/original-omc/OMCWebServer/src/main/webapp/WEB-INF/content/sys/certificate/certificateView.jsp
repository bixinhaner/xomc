<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
	.slide-content{
		margin-left:50px;
	}
	.licEdit .el-textarea{
		width:300px;
	}
	.infoWarp{
		padding:20px 10px; 
		width:98%;
		display:flex;
	}
	
	.infoWarp .info label{
		color:#999999;
		margin-right:5px;
		text-align:left;
		display:inline-block;
		width:22%;
		float:left;		
	}
	.infoWarp .info{
		font-size:12px;
		width:100%;
		position:relative;
		display:inline-block;
		padding-bottom:16px;
	}
	.infoWarp .info span{
		color:#333333;
		width:74%;
		float:right;
	}
	.infoTitle{
		font-size:14px;
		font-weight:bold;
		color:#333333;
		border-top:2px;
		margin-left:-10px;
		padding:10px 0 16px;
	}
	.issueNoColor:before{		
		color:#CFCFCF;
	}
	.issueOkColor:before{
		color:#67D972;
	}
	.issueErrorColor:before,
	.statusError:before{
		color:#E88282;
	}
	.caInfo{
		width:43%; 
		height:auto;
		border:1px solid #EAEAEA; 
		border-top:2px solid #5388FF;
		border-radius:2px;
		margin-right:20px;
		padding:0 26px 10px;
	}
	.ipsecInfo{
		width:43%;
		height:auto; 
		border:1px solid #EAEAEA;
		border-radius:2px;
		border-top:2px solid #5388FF;
		padding:0 26px 10px;
	}
	#viewCertificate .issueInfo{
		margin-right:8px;
		font-size:18px;
		float:left;		
	}
	.issueInfoInpro{
		width:18px;
		height:18px;
		position:absolute;
		top:0;
		float:left;	
		margin-top:0;
	}

	.issueInpro{
		margin-right:8px;
		font-size:18px;
		float:left;
	}
	.issueName{
		margin-left:30px;
	}
	.description{
		float:right;
		line-height:24px;
		width:74%;		
	}
	.fileName{
		position:relative;
	}
	.infoTip{
		color:#666;	
		font-weight:normal;
		font-size:14px;
		margin-left: 6px;
	}
	.el-card__body{
		border:none !important;
	}
	.slidebarTitleDiv{
		border-bottom:1px solid #E9E9E9;
	}
</style>

<div id='viewCertificate'>
	<div class="slidebarTitleDiv">
		<div class="slideTitle"><%=rb.getString("XinXi")%> <span v-html="serialNumber" class="infoTip"></span></div>
		
		<div class="el-icon el-icon-circle-close" @click="cancelSlide" style="position:absolute;right:20px;top:10px;"></div>
	</div>
	<div class="infoWarp">
		<div class="caInfo">
			<p class="infoTitle"><%=rb.getString("CAZhengShu")%></p>
			<div class="info">
				<label><%=rb.getString("CAZhengShu")%></label>
				<span class="fileName" v-html="caFileName"></span>
			</div>
			<div class="info">
				<label><%=rb.getString("ShangChuanRen")%></label>
				<span>{{caUploader}}</span>
			</div>
			<div class="info">
				<label><%=rb.getString("ShangChuanShiJian")%></label>
				<span>{{caUploadTime}}</span>
			</div>
			<div class="info">
				<label><%=rb.getString("XiaFaShiJian")%></label>
				<span>{{caDownTime}}</span>
			</div>
			<%-- <div class="info">
				<label><%=rb.getString("ShengXiaoGengXinShiJian")%></label>
				<span>{{updateTime}}</span>
			</div>
			<div class="info">
				<label><%=rb.getString("ZSShengXiaoZhuangTai")%></label>
				<span v-html="enableStatus"></span>
			</div> --%>
			<div class="info">
				<label><%=rb.getString("WangGuanMiaoShu")%></label>
				<p class="description">{{caDescription}}</p>
			</div>
		</div>
		<div class="ipsecInfo">
			<p class="infoTitle"><%=rb.getString("SheBeiZhengShu")%></p>
			<div class="info">
				<label><%=rb.getString("IpsecZhengShu")%></label>
				<span class="fileName" v-html="certFileName"></span>
			</div>
			<div class="info">
				<label><%=rb.getString("MiYaoZhengShu")%></label>
				<span class="fileName" v-html="secretKeyFileName"></span>
			</div>
			<div class="info">
				<label><%=rb.getString("ShangChuanRen")%></label>
				<span>{{privateUploader}}</span>
			</div>
			<div class="info">
				<label><%=rb.getString("ShangChuanShiJian")%></label>
				<span>{{privateUploadTime}}</span>
			</div>
			<div class="info">
				<label><%=rb.getString("XiaFaShiJian")%></label>
				<span>{{privateDownTime}}</span>
			</div>
			<%-- <div class="info">
				<label><%=rb.getString("ShengXiaoGengXinShiJian")%></label>
				<span>{{updateTime}}</span>
			</div>
			<div class="info">
				<label><%=rb.getString("ZSShengXiaoZhuangTai")%></label>
				<span v-html="enableStatus"></span>
			</div> --%>
			<div class="info">
				<label><%=rb.getString("ZhengShuMiaoShu")%></label>
				<p class="description">{{privateDescription}}</p>
			</div>
		</div>	
	</div>
</div>
<script>
	new Vue({
		el:'#viewCertificate',
		data(){
			return{				
				'caFileName':'',
				'caUploader':'',				
				'caUploadTime':'',
				'caDownTime':'',
				'caDescription':'',
				//'updateTime':'',
				//'enableStatus':'',				
				'certFileName':'',
				'secretKeyFileName':'',
				'privateUploader':'',
				'privateUploadTime':'',
				'privateDownTime':'',
				'privateDescription':'',
				serialNumber:''
			}
		},
		methods:{
			init(fileId,serialNumber){
				var vm = this;	
				vm.serialNumber = '(<%=rb.getString("XiaoZhanBianMa")%>' + serialNumber + '）';
				axios.post('${ctx}/cell/cert/getIpsecCertInfoById.action',stringify({
					id : fileId,
					timeZone: timeZone
				})).then(function(response){
					var data = response.data;				
					/* var data = {
							"backupFileName": '',
							"backupUploadStatus": null,
							"caFileName": "考虑开发和",
							"caUploader":"我自己",
							"caDownTime":"2020-11-01 08:08:08",								
							"caUploadStatus": "3",
							"certFileName": "privateCert (4).tar.gz",
							"certUploadStatus": "3",
							"caDescription": "告诉股份萨科技股份来空间撒过分了就开始噶付款及挨个看见过",
							"enableStatus": "2",
							"fileId": null,
							"id": 20,
							"secretKeyFileName": "privateCert (6).tar.gz",
							"secretKeyUploadStatus": "3",
							"serialNumber": "12150000012043B0010",
							"updateTime": "2020-11-01 08:08:08",

							"privateDescription":"DHSIUYIUSAG和空间几点上课讲话的空间上的讲话撒离开大家哈时间看到哈市了空间很大空间撒恢复健康的十分快结婚的时刻就恢复快决定是否可决定是否可见还是打开就发货速度快解放和会计师的很费劲的恢复健康活动DIUGSDAIUGSU",
							"privateUploadTime":"2020-11-01 08:08:08",
							"privateDownTime":"2020-11-01 08:08:08",
							"caUploadTime":"2020-11-01 08:08:08",
							"privateUploader":"admin"
					}  */ 
					
					vm.caUploader = data.caUploader;
					vm.caUploadTime = data.caUploadTime;
					vm.caDownTime = data.caDownTime;//ca 证书 下发时间				
					vm.caDescription = data.caDescription;					
					//vm.updateTime = data.updateTime;//生效更新时间					
					vm.privateUploader = data.privateUploader;
					vm.privateUploadTime = data.privateUploadTime;
					vm.privateDownTime = data.privateDownTime;					
					vm.privateDescription = data.privateDescription;
					
					//0 == 下发失败，1 == 已下发，2 == 下发中，3 == 未下发 --> 
					//如果证书为空时
					if(data.caFileName == '' || data.caFileName == null || data.caFileName == undefined){
						vm.caStatus = '';					
					}else{
						if(data.caUploadStatus == '0'){
							vm.caFileName = '<i class="el-icon el-icon-status-issued issueErrorColor issueInfo"></i>'+data.caFileName;					
						}else if(data.caUploadStatus == '1'){
							vm.caFileName = '<i class="el-icon el-icon-status-issued issueOkColor issueInfo"></i>'+data.caFileName;
						}else if(data.caUploadStatus == '2'){
							vm.caFileName = '<i class="status_issuing issueInfoInpro"></i><p class="issueName">'+data.caFileName+'</p>';
						}else if(data.caUploadStatus == '3'){
							vm.caFileName = '<i class="el-icon el-icon-status-issued issueNoColor issueInfo"></i>'+data.caFileName;
						}
					}
					// #41930 7.0.2版本将证书生效功能删除
					//生效状态
					<%-- if(data.enableStatus == '0'){
						vm.enableStatus = '<i class="el-icon el-icon-status-failed-take-effect statusError issueInfo"></i>'+'<%=rb.getString("ShengXiaoShiBai")%>';					
					}else if(data.enableStatus == '1'){
						vm.enableStatus = '<i class="el-icon el-icon-status-take-effect issueOkColor issueInfo"></i>'+'<%=rb.getString("YiShengXiao")%>';
					}else if(data.enableStatus == '2'){
						vm.enableStatus = '<i class="status_issuing issueInfoInpro"></i><p class="issueName">'+'<%=rb.getString("ShengXiaoZhong")%>';
					}else if(data.enableStatus == '3'){
						vm.enableStatus = '<i class="el-icon el-icon-status-waitting-take-effect issueInpro"></i>'+'<%=rb.getString("DaiShengXiao")%>';
					} --%>
					//ipsec 证书
					if(data.certFileName == '' || data.certFileName == null || data.certFileName == undefined){
						vm.certFileName = '';					
					}else{					
						if(data.certUploadStatus == '0'){
							vm.certFileName = '<i class="el-icon el-icon-status-issued issueErrorColor issueInfo"></i>'+data.certFileName;					
						}else if(data.certUploadStatus == '1'){
							vm.certFileName = '<i class="el-icon el-icon-status-issued issueOkColor issueInfo"></i>'+data.certFileName;
						}else if(data.certUploadStatus == '2'){
							vm.certFileName = '<i class="status_issuing issueInfoInpro"></i><p class="issueName">'+data.certFileName+'</p>';
						}else if(data.certUploadStatus == '3'){
							vm.certFileName = '<i class="el-icon el-icon-status-issued issueNoColor issueInpro"></i>'+data.certFileName;
						}
					}
					
					//秘钥 证书
					if(data.secretKeyFileName == '' || data.secretKeyFileName == null || data.secretKeyFileName == undefined){
						vm.secretKeyFileName = '';					
					}else{
						
						if(data.secretKeyUploadStatus == '0'){
							vm.secretKeyFileName = '<i class="el-icon el-icon-status-issued issueErrorColor issueInfo"></i>'+data.secretKeyFileName;					
						}else if(data.secretKeyUploadStatus == '1'){
							vm.secretKeyFileName = '<i class="el-icon el-icon-status-issued issueOkColor issueInfo"></i>'+data.secretKeyFileName;
						}else if(data.secretKeyUploadStatus == '2'){
							vm.secretKeyFileName = '<i class="status_issuing issueInfoInpro"></i><p class="issueName">'+data.secretKeyFileName+'</p>';
						}else if(data.secretKeyUploadStatus == '3'){
							vm.secretKeyFileName = '<i class="el-icon el-icon-status-issued issueNoColor issueInpro"></i>'+data.secretKeyFileName;
						}
					}
				}) 	
			},
			cancelSlide(){
				eventBus.$emit('close-certificateView')	
			},
		},
		mounted(){
			eventBus.$off("to-cetInfo").$on("to-cetInfo",this.init);
		}
	})
</script>
