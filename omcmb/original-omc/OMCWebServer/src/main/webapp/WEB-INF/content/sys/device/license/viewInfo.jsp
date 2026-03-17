<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
	.licView{
		width:500px;
		margin-bottom:20px;
	}
	.licView label{
		display:inline-block;
		width:140px;
		text-align:right;
		margin-bottom:8px;
		font-size:14px;
		color:#5A7B92;
	}
	.licView span{
		font-size:14px;
		color:#333;
	}
	.slide-content{
		margin-left:50px;
	}
	.licEdit .el-textarea{
		width:300px;
	}
</style>
<div id='viewLicense' style='padding-top:20px;'>
	<div class='licView'>
		<label>1588 License File&nbsp;&nbsp;:&nbsp;&nbsp;</label>
		<span v-html='file_name'></span>
	</div>
	<div class='licView'>
		<label><%=rb.getString("MACDiZhi")%>&nbsp;&nbsp;:&nbsp;&nbsp;</label>
		<span style='display:inline-block;width:400px;word-wrap:break-word;margin-left:145px;margin-top:-27px;' v-html='mac_range'></span>
	</div>
	<div class='licView'>
		<label><%=rb.getString("WenJianDaXiao")%>&nbsp;&nbsp;:&nbsp;&nbsp;</label>
		<span v-html='file_size'></span>
	</div>
	<div class='licView'>
		<label><%=rb.getString("ShangChuanRen")%>&nbsp;&nbsp;:&nbsp;&nbsp;</label>
		<span v-html='uploader'></span>
	</div>
	<div class='licView'>
		<label><%=rb.getString("ShangChuanShiJian")%>&nbsp;&nbsp;:&nbsp;&nbsp;</label>
		<span v-html='upload_time'></span>
	</div>
	<div class='licView'>
		<label><%=rb.getString("MiaoShu")%>&nbsp;&nbsp;:&nbsp;&nbsp;</label>
		<span v-html='description'></span>
	</div>
</div>
<script>
	new Vue({
		el:'#viewLicense',
		data(){
			return{
				file_name:'',
				mac_range:'',
				file_size:'',
				description:'',
				uploader:'',
				upload_time:'',
				defaultDesc:'',
				file_id:''
			}
		},
		methods:{
			getFileInfo(file_id){
				var vm = this;
				vm.file_id = file_id;
				axios.post('${ctx}/cell/1588License/getFileInfoById.action',stringify({
					file_id : file_id,
					time_zone　: timeZone
				})).then(function(response){
					var data = response.data.rows[0];
					vm.file_name = data.file_name;
					vm.mac_range = data.mac_section.replace(/,/g,',<br/>');
					vm.file_size = data.file_size;
					vm.description = data.description;
					vm.uploader = data.uploader;
					vm.upload_time = data.upload_time;
				})
			}
		},
		mounted(){
			eventBus.$off('view-file').$on('view-file',this.getFileInfo)
		}
	})
</script>
